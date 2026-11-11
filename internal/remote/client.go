// Package remote is a client for the `goh serve` local HTTP API, used by
// `goh attach` to drive a running server from the terminal UI. The API is
// loopback only, so the client and server share a machine.
package remote

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/localapi"
	"github.com/Sanidhya14321/goh/internal/session"
)

// Client talks to one server.
type Client struct {
	base  *url.URL
	token string
	http  *http.Client
}

// ErrUncertain reports that the session has interrupted tool effects the
// server will not continue past without an explicit acknowledgment.
var ErrUncertain = errors.New("the session has uncertain tool effects; inspect the project, then attach with --acknowledge-uncertain")

// New checks that address is a numeric loopback http URL (as `goh serve`
// listens on) and returns a client that authenticates with token.
func New(address, token string) (*Client, error) {
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "http" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || !localapi.IsLoopbackHost(u.Host) {
		return nil, errors.New("attach needs the server's numeric loopback address, such as http://127.0.0.1:4096")
	}
	if token == "" {
		return nil, errors.New("attach needs the server token (the HARNESS_SERVER_TOKEN the server was started with)")
	}
	u.Path = ""
	// No overall timeout: a run streams for as long as it works. Redirects
	// are refused so the token never goes anywhere else.
	h := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &Client{base: u, token: token, http: h}, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base.String()+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

// call makes a short request and decodes a JSON reply into out (when not nil).
func (c *Client) call(ctx context.Context, method, path string, body, out any, want int) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		return serverError(resp)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}

func serverError(resp *http.Response) error {
	var e struct {
		Error   string   `json:"error"`
		Pending []string `json:"pending"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&e)
	if len(e.Pending) > 0 {
		return ErrUncertain
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("the server refused the token")
	}
	if e.Error == "" {
		e.Error = resp.Status
	}
	return fmt.Errorf("server: %s", oneLine(e.Error))
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)), " ")
	if r := []rune(s); len(r) > 300 {
		s = string(r[:300]) + "…"
	}
	return s
}

// Info describes the server.
func (c *Client) Info(ctx context.Context) (localapi.Info, error) {
	var info localapi.Info
	err := c.call(ctx, http.MethodGet, "/v1/info", nil, &info, http.StatusOK)
	return info, err
}

// Sessions lists the server project's sessions, newest first.
func (c *Client) Sessions(ctx context.Context) ([]session.Info, error) {
	var all []session.Info
	after := ""
	for range 50 { // at most 5000 sessions
		var page struct {
			Sessions []session.Info `json:"sessions"`
			Next     string         `json:"next_after"`
		}
		path := "/v1/sessions?limit=100"
		if after != "" {
			path += "&after=" + url.QueryEscape(after)
		}
		if err := c.call(ctx, http.MethodGet, path, nil, &page, http.StatusOK); err != nil {
			return nil, err
		}
		all = append(all, page.Sessions...)
		if page.Next == "" {
			break
		}
		after = page.Next
	}
	return all, nil
}

// Create starts a new session on the server.
func (c *Client) Create(ctx context.Context) (string, error) {
	var info session.Info
	if err := c.call(ctx, http.MethodPost, "/v1/sessions", nil, &info, http.StatusCreated); err != nil {
		return "", err
	}
	return info.ID, nil
}

// Handlers receive what a run streams: events, and requests for input that
// must be answered before the run continues.
type Handlers struct {
	Emit    func(core.Event)
	Approve func(context.Context, core.Permission) (core.Decision, error)
	Ask     func(context.Context, core.QuestionRequest) (core.QuestionReply, error)
}

// Run sends prompt to session and relays the run until it ends. Cancelling
// ctx asks the server to cancel the run and waits for it to finish. The
// returned status is the run's own (completed, cancelled, failed and so on).
func (c *Client) Run(ctx context.Context, sessionID, prompt string, acknowledge bool, h Handlers) (string, error) {
	body := map[string]any{"prompt": prompt}
	if acknowledge {
		body["acknowledge_uncertain"] = true
	}
	// The stream outlives ctx so a cancelled run can still report its end.
	stream, stop := context.WithCancel(context.Background())
	defer stop()
	resp, err := c.do(stream, http.MethodPost, "/v1/sessions/"+url.PathEscape(sessionID)+"/run", body)
	if err != nil {
		return "failed", fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "failed", serverError(resp)
	}
	runID := resp.Header.Get("X-Harness-Run-ID")
	cancelled := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if runID != "" {
				_ = c.call(context.Background(), http.MethodPost, "/v1/runs/"+url.PathEscape(runID)+"/cancel", nil, nil, http.StatusAccepted)
			}
			// Stop reading if the server does not end the stream soon.
			select {
			case <-time.After(30 * time.Second):
				stop()
			case <-cancelled:
			}
		case <-cancelled:
		}
	}()
	defer close(cancelled)

	status, finalErr := "failed", errors.New("the server ended the run without a result")
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64<<10), 16<<20)
	for scanner.Scan() {
		var frame struct {
			Event *core.Event `json:"event"`
			Input *struct {
				ID   string          `json:"id"`
				Kind string          `json:"kind"`
				Data json.RawMessage `json:"data"`
			} `json:"input"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &frame); err != nil {
			return "failed", errors.New("the server sent an invalid stream frame")
		}
		switch {
		case frame.Event != nil:
			e := normalize(*frame.Event)
			if runID == "" {
				runID = e.RunID
			}
			if e.Type == "run_finished" {
				var d struct {
					Status string `json:"status"`
					Error  string `json:"error"`
				}
				b, _ := json.Marshal(e.Data)
				_ = json.Unmarshal(b, &d)
				status, finalErr = d.Status, nil
				if d.Error != "" {
					finalErr = errors.New(oneLine(d.Error))
				}
			}
			if h.Emit != nil {
				h.Emit(e)
			}
		case frame.Input != nil:
			reply, err := c.answer(ctx, frame.Input.Kind, frame.Input.Data, h)
			if err != nil {
				return "failed", err
			}
			path := "/v1/runs/" + url.PathEscape(runID) + "/inputs/" + url.PathEscape(frame.Input.ID)
			if err := c.call(context.Background(), http.MethodPost, path, reply, nil, http.StatusNoContent); err != nil {
				return "failed", err
			}
		}
	}
	if err := scanner.Err(); err != nil && finalErr != nil {
		return "failed", fmt.Errorf("lost the server stream: %w", err)
	}
	if status == "" {
		status = "failed"
	}
	return status, finalErr
}

// answer asks the local user for a permission or question reply. A
// cancelled or dismissed prompt is sent as a denial or dismissal, so the
// server's run is never left waiting.
func (c *Client) answer(ctx context.Context, kind string, data json.RawMessage, h Handlers) (any, error) {
	switch kind {
	case "permission":
		var p core.Permission
		if err := json.Unmarshal(data, &p); err != nil || h.Approve == nil {
			return map[string]bool{"allow": false}, nil
		}
		d, err := h.Approve(ctx, p)
		return map[string]bool{"allow": err == nil && d.Allow}, nil
	case "question":
		var q core.QuestionRequest
		if err := json.Unmarshal(data, &q); err != nil || h.Ask == nil {
			return map[string]bool{"dismiss": true}, nil
		}
		reply, err := h.Ask(ctx, q)
		if err != nil {
			return map[string]bool{"dismiss": true}, nil
		}
		return map[string]any{"answers": reply.Answers}, nil
	}
	return nil, fmt.Errorf("the server asked for unknown input %q", oneLine(kind))
}

// normalize restores typed payloads that interfaces type-assert; other
// events are read through JSON already.
func normalize(e core.Event) core.Event {
	if e.Type == "provider_retry" {
		var n core.RetryNotice
		b, _ := json.Marshal(e.Data)
		if json.Unmarshal(b, &n) == nil {
			e.Data = n
		}
	}
	return e
}
