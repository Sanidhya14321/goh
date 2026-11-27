package webfetch

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sanidhya14321/goh/internal/platform"
)

// Rendering runs a headless Chromium-family browser on the page, with every
// connection it makes (the page, its scripts, images, XHR) sent through a
// loopback proxy that applies webfetch's address rule: the browser gets no
// route to loopback, private-network or link-local addresses unless
// webfetch_allow_private is set. The proxy resolves names itself, so DNS
// rebinding is checked at the connected address, as for plain fetches.

const (
	renderSettle  = 8000 // ms of virtual time for scripts to build the page
	maxProxyConns = 64
	proxyIdle     = 30 * time.Second
)

// browserCandidates are where FindBrowser looks for "auto".
func browserCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"}
	case "windows":
		var out []string
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData")} {
			if base != "" {
				out = append(out, filepath.Join(base, `Google\Chrome\Application\chrome.exe`), filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`))
			}
		}
		return out
	}
	var out []string
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge"} {
		if p, err := exec.LookPath(name); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// FindBrowser resolves the webfetch_browser setting: "auto" finds an
// installed Chrome, Chromium, Edge or Brave; anything else must be an
// absolute path to an executable.
func FindBrowser(setting string) (string, error) {
	if setting == "auto" {
		for _, p := range browserCandidates() {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p, nil
			}
		}
		return "", errors.New("webfetch_browser is auto, but no Chrome, Chromium, Edge or Brave was found; set its absolute path")
	}
	if !filepath.IsAbs(setting) {
		return "", errors.New(`webfetch_browser must be "auto" or an absolute path to a Chromium-family browser`)
	}
	if info, err := os.Stat(setting); err != nil || info.IsDir() {
		return "", fmt.Errorf("webfetch_browser %s is not an executable file", setting)
	}
	return setting, nil
}

// SetBrowser enables render with the browser at path ("" disables it).
func (c *Client) SetBrowser(path string) { c.browser = path }

// CanRender reports whether render is available.
func (c *Client) CanRender() bool { return c != nil && c.browser != "" }

// proxy is a minimal HTTP proxy (CONNECT and absolute-URI requests) whose
// every outbound connection goes through the address-checking dialer.
type proxy struct {
	listener net.Listener
	dial     func(ctx context.Context, network, address string) (net.Conn, error)
	ctx      context.Context
	conns    atomic.Int32
	refused  atomic.Int32
	wg       sync.WaitGroup
	mu       sync.Mutex
	open     map[net.Conn]bool
}

func newProxy(ctx context.Context, allowPrivate bool) (*proxy, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	if !allowPrivate {
		dialer.Control = publicOnly
	}
	p := &proxy{listener: l, dial: dialer.DialContext, ctx: ctx, open: map[net.Conn]bool{}}
	p.wg.Add(1)
	go p.serve()
	return p, nil
}

func (p *proxy) address() string { return p.listener.Addr().String() }

func (p *proxy) track(c net.Conn, add bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if add {
		p.open[c] = true
	} else {
		delete(p.open, c)
	}
}

func (p *proxy) close() {
	p.listener.Close()
	p.mu.Lock()
	for c := range p.open {
		c.Close()
	}
	p.mu.Unlock()
	p.wg.Wait()
}

func (p *proxy) serve() {
	defer p.wg.Done()
	for {
		c, err := p.listener.Accept()
		if err != nil {
			return
		}
		if p.conns.Add(1) > maxProxyConns {
			p.conns.Add(-1)
			c.Close()
			continue
		}
		p.track(c, true)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			defer p.conns.Add(-1)
			defer p.track(c, false)
			defer c.Close()
			p.handle(c)
		}()
	}
}

func (p *proxy) handle(client net.Conn) {
	client.SetReadDeadline(time.Now().Add(proxyIdle))
	reader := bufio.NewReaderSize(client, 16<<10)
	// The browser is the only expected client; the deadline bounds a slow
	// or oversized header from anything else on this machine.
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}
	client.SetReadDeadline(time.Time{})
	if req.Method == http.MethodConnect {
		upstream, err := p.dial(p.ctx, "tcp", req.Host)
		if err != nil {
			p.refused.Add(1)
			io.WriteString(client, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n")
			return
		}
		p.track(upstream, true)
		defer p.track(upstream, false)
		defer upstream.Close()
		io.WriteString(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
		done := make(chan struct{}, 2)
		go func() { io.Copy(upstream, reader); closeWrite(upstream); done <- struct{}{} }()
		go func() { io.Copy(client, upstream); closeWrite(client); done <- struct{}{} }()
		<-done
		<-done
		return
	}
	if req.URL == nil || req.URL.Scheme != "http" || req.URL.Host == "" {
		io.WriteString(client, "HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n")
		return
	}
	req.Body = io.NopCloser(io.LimitReader(reader, req.ContentLength))
	if req.ContentLength < 0 {
		req.Body = http.NoBody
	}
	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authorization")
	transport := &http.Transport{Proxy: nil, DialContext: p.dial, ResponseHeaderTimeout: 30 * time.Second, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	resp, err := transport.RoundTrip(req.WithContext(p.ctx))
	if err != nil {
		p.refused.Add(1)
		io.WriteString(client, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
		return
	}
	defer resp.Body.Close()
	resp.Close = true
	resp.Write(client)
}

func closeWrite(c net.Conn) {
	if tc, ok := c.(interface{ CloseWrite() error }); ok {
		tc.CloseWrite()
	}
}

// cappedBuffer keeps at most limit bytes and records that more arrived.
type cappedBuffer struct {
	bytes.Buffer
	limit int
	over  bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.Len(); len(p) > room {
		b.over = true
		if room > 0 {
			b.Buffer.Write(p[:room])
		}
		return len(p), nil
	}
	return b.Buffer.Write(p)
}

// render loads target in the browser and returns the page's DOM after its
// scripts ran, as HTML.
func (c *Client) render(ctx context.Context, target string) ([]byte, error) {
	if !c.CanRender() {
		return nil, errors.New("webfetch render needs webfetch_browser set to \"auto\" or a Chrome path")
	}
	p, err := newProxy(ctx, c.allowPrivate)
	if err != nil {
		return nil, fmt.Errorf("webfetch render could not start its proxy: %w", err)
	}
	defer p.close()
	profile, err := os.MkdirTemp("", "goh-render-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(profile)
	args := []string{"--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check",
		"--disable-extensions", "--disable-background-networking", "--disable-sync", "--disable-default-apps",
		"--disable-component-update", "--disable-quic", "--dns-prefetch-disable", "--mute-audio", "--hide-scrollbars",
		"--force-webrtc-ip-handling-policy=disable_non_proxied_udp", "--webrtc-ip-handling-policy=disable_non_proxied_udp",
		"--password-store=basic", "--use-mock-keychain",
		"--user-data-dir=" + profile,
		"--proxy-server=http://" + p.address(),
		"--proxy-bypass-list=<-loopback>", // loopback goes through the proxy too
		fmt.Sprintf("--virtual-time-budget=%d", renderSettle),
		"--dump-dom", target}
	cmd := exec.Command(c.browser, args...)
	// Only what the browser needs: no proxy variables, no credentials.
	cmd.Env = []string{"HOME=" + profile, "TMPDIR=" + profile, "PATH=" + os.Getenv("PATH")}
	if runtime.GOOS == "windows" {
		cmd.Env = append(cmd.Env, "SystemRoot="+os.Getenv("SystemRoot"), "LOCALAPPDATA="+profile, "TEMP="+profile, "TMP="+profile)
	}
	stdout := &cappedBuffer{limit: MaxResponse + 1}
	stderr := &cappedBuffer{limit: 8 << 10}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	platform.NewGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("webfetch render could not start the browser: %w", err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	select {
	case err = <-exited:
	case <-ctx.Done():
		platform.KillGroup(cmd.Process.Pid)
		<-exited
		return nil, errors.New("webfetch render timed out or was cancelled")
	}
	platform.KillGroup(cmd.Process.Pid) // helpers the browser left behind
	if stdout.over {
		return nil, errors.New("webfetch response exceeds 4MiB")
	}
	page := bytes.TrimSpace(stdout.Bytes())
	if err != nil || len(page) == 0 {
		detail := strings.TrimSpace(stderr.String())
		if i := strings.LastIndexByte(detail, '\n'); i >= 0 {
			detail = detail[i+1:]
		}
		if i := strings.Index(detail, "] "); strings.HasPrefix(detail, "[") && i > 0 {
			detail = detail[i+2:] // Chrome's log prefix: [pid:tid:time:LEVEL:file]
		}
		if p.refused.Load() > 0 {
			return nil, fmt.Errorf("webfetch render failed; %d connection(s) to non-public addresses or unreachable hosts were refused", p.refused.Load())
		}
		return nil, fmt.Errorf("webfetch render failed: the browser returned no page (%s)", clipText(detail, 200))
	}
	return page, nil
}
