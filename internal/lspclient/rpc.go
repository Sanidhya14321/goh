package lspclient

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxFrame = 1 << 20

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return fmt.Sprintf("LSP RPC error %d: %s", e.Code, e.Message) }

type rpcClient struct {
	reader      io.ReadCloser
	writer      io.WriteCloser
	writeMu     sync.Mutex
	mu          sync.Mutex
	next        uint64
	pending     map[uint64]chan rpcMessage
	done        chan struct{}
	once        sync.Once
	err         error
	diagnostics map[string]json.RawMessage
	// diagVersion is the document version a server attached to the
	// diagnostics it published (0 when it attached none).
	diagVersion map[string]int
	// diagAt is when each uri's diagnostics arrived.
	diagAt      map[string]time.Time
	changed     chan struct{}
	serverSlots chan struct{}
	settings    map[string]any
}

func newRPC(reader io.ReadCloser, writer io.WriteCloser, settings map[string]any) *rpcClient {
	c := &rpcClient{serverSlots: make(chan struct{}, 8), reader: reader, writer: writer, pending: map[uint64]chan rpcMessage{}, done: make(chan struct{}), diagnostics: map[string]json.RawMessage{}, diagVersion: map[string]int{}, diagAt: map[string]time.Time{}, changed: make(chan struct{}, 1), settings: settings}
	go c.readLoop()
	return c
}
func readFrame(r *bufio.Reader) ([]byte, error) {
	size := -1
	headerBytes := 0
	for {
		var buffer []byte
		for {
			part, err := r.ReadSlice('\n')
			headerBytes += len(part)
			if headerBytes > 8192 {
				return nil, errors.New("LSP header exceeds 8KiB")
			}
			buffer = append(buffer, part...)
			if err == bufio.ErrBufferFull {
				continue
			}
			if err != nil {
				return nil, err
			}
			break
		}
		line := string(buffer)
		if !strings.HasSuffix(line, "\r\n") {
			return nil, errors.New("LSP header requires CRLF framing")
		}
		line = strings.TrimSuffix(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, errors.New("malformed LSP header")
		}
		if strings.EqualFold(name, "Content-Length") {
			if size != -1 {
				return nil, errors.New("duplicate LSP Content-Length")
			}
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			size = parsed
			if err != nil || size < 1 || size > maxFrame {
				return nil, errors.New("LSP Content-Length must be 1..1048576 bytes")
			}
		}
	}
	if size < 1 {
		return nil, errors.New("missing LSP Content-Length")
	}
	data := make([]byte, size)
	_, err := io.ReadFull(r, data)
	return data, err
}
func (c *rpcClient) readLoop() {
	reader := bufio.NewReaderSize(c.reader, 8192)
	for {
		data, err := readFrame(reader)
		if err != nil {
			c.fail(fmt.Errorf("LSP connection closed: %w", err))
			return
		}
		var message rpcMessage
		if err := json.Unmarshal(data, &message); err != nil || message.JSONRPC != "2.0" {
			c.fail(errors.New("invalid LSP JSON-RPC message"))
			return
		}
		if message.Method != "" {
			if len(message.ID) > 0 {
				select {
				case c.serverSlots <- struct{}{}:
					go func() { defer func() { <-c.serverSlots }(); c.serverRequest(message) }()
				default:
					c.fail(errors.New("LSP server exceeded 8 concurrent client requests"))
					return
				}
				continue
			}
			if message.Method == "textDocument/publishDiagnostics" {
				var params struct {
					URI         string          `json:"uri"`
					Version     int             `json:"version"`
					Diagnostics json.RawMessage `json:"diagnostics"`
				}
				if json.Unmarshal(message.Params, &params) == nil && params.URI != "" && len(params.Diagnostics) > 0 {
					if len(params.Diagnostics) > 64<<10 || len(params.URI) > 8192 {
						c.fail(errors.New("LSP diagnostics exceed bounded payload"))
						return
					}
					c.mu.Lock()
					if len(c.diagnostics) < 64 || c.diagnostics[params.URI] != nil {
						c.diagnostics[params.URI] = params.Diagnostics
						c.diagVersion[params.URI] = params.Version
						c.diagAt[params.URI] = time.Now()
					}
					c.mu.Unlock()
					select {
					case c.changed <- struct{}{}:
					default:
					}
				}
			}
			continue
		}
		if len(message.ID) == 0 {
			c.fail(errors.New("LSP response missing ID"))
			return
		}
		var id uint64
		if json.Unmarshal(message.ID, &id) != nil {
			c.fail(errors.New("LSP response has invalid request ID"))
			return
		}
		c.mu.Lock()
		response := c.pending[id]
		delete(c.pending, id)
		c.mu.Unlock()
		if response != nil {
			response <- message
		}
	}
}
func (c *rpcClient) serverRequest(message rpcMessage) {
	// Language servers cannot apply edits, run commands, or expand file access.
	response := rpcMessage{JSONRPC: "2.0", ID: message.ID}
	switch message.Method {
	case "window/workDoneProgress/create":
		response.Result = json.RawMessage("null")
	case "workspace/configuration":
		var params struct {
			Items []struct {
				Section string `json:"section"`
			} `json:"items"`
		}
		if json.Unmarshal(message.Params, &params) != nil {
			response.Error = &rpcError{-32602, "invalid configuration request"}
			break
		}
		values := make([]any, len(params.Items))
		for i, item := range params.Items {
			var value any = c.settings
			for _, key := range strings.Split(item.Section, ".") {
				if item.Section == "" {
					break
				}
				m, ok := value.(map[string]any)
				if !ok {
					value = nil
					break
				}
				value = m[key]
			}
			values[i] = value
		}
		response.Result, _ = json.Marshal(values)
	default:
		response.Error = &rpcError{-32601, "client operation unsupported; server-requested effects are prohibited"}
	}
	if err := c.write(response); err != nil {
		c.fail(err)
	}
}

// settleServerRequests waits, up to limit, for replies to server-initiated
// requests (refusals included) to be written before shutdown begins.
func (c *rpcClient) settleServerRequests(limit time.Duration) {
	deadline := time.Now().Add(limit)
	for len(c.serverSlots) > 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
}
func (c *rpcClient) write(message rpcMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if len(data) > maxFrame {
		return errors.New("outbound LSP frame exceeds 1 MiB")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	select {
	case <-c.done:
		return c.failure()
	default:
	}
	frame := fmt.Appendf(nil, "Content-Length: %d\r\n\r\n", len(data))
	frame = append(frame, data...)
	n, err := c.writer.Write(frame)
	if err == nil && n != len(frame) {
		err = io.ErrShortWrite
	}
	if err != nil {
		c.fail(err)
	}
	return err
}
func (c *rpcClient) notify(method string, params any) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return c.write(rpcMessage{JSONRPC: "2.0", Method: method, Params: encoded})
}
func (c *rpcClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, c.failure()
	default:
	}
	c.mu.Lock()
	if len(c.pending) >= 64 {
		c.mu.Unlock()
		return nil, errors.New("LSP pending request limit reached")
	}
	c.next++
	id := c.next
	response := make(chan rpcMessage, 1)
	c.pending[id] = response
	c.mu.Unlock()
	defer func() { c.mu.Lock(); delete(c.pending, id); c.mu.Unlock() }()
	rawID, _ := json.Marshal(id)
	// Writes may block when a peer stops reading. The owner closes the pipe on
	// cancellation, so every blocked writer is released before process teardown.
	written := make(chan error, 1)
	go func() { written <- c.write(rpcMessage{JSONRPC: "2.0", ID: rawID, Method: method, Params: encoded}) }()
	select {
	case err := <-written:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		go c.notify("$/cancelRequest", map[string]any{"id": id})
		return nil, ctx.Err()
	case <-c.done:
		return nil, c.failure()
	}
	select {
	case result := <-response:
		if result.Error != nil {
			return nil, result.Error
		}
		if result.Result == nil {
			return nil, errors.New("LSP response missing result")
		}
		return result.Result, nil
	case <-ctx.Done():
		go c.notify("$/cancelRequest", map[string]any{"id": id})
		return nil, ctx.Err()
	case <-c.done:
		return nil, c.failure()
	}
}
func (c *rpcClient) fail(err error) {
	c.once.Do(func() {
		c.mu.Lock()
		c.err = err
		c.mu.Unlock()
		close(c.done)
		c.reader.Close()
		c.writer.Close()
	})
}
func (c *rpcClient) failure() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	return errors.New("LSP connection closed")
}

// cachedDiagnostics returns diagnostics published for uri, skipping ones a
// server marked as being for a version older than version. quiet is how long
// ago unversioned diagnostics arrived (zero for versioned ones): those may
// describe the previous text, so the caller lets them settle.
func (c *rpcClient) cachedDiagnostics(uri string, version int) (data json.RawMessage, ok bool, unversioned bool, quiet time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok = c.diagnostics[uri]
	v := c.diagVersion[uri]
	if ok && v != 0 && v < version {
		return nil, false, false, 0
	}
	if ok && v == 0 {
		unversioned, quiet = true, time.Since(c.diagAt[uri])
	}
	return append(json.RawMessage(nil), data...), ok, unversioned, quiet
}

// forgetDiagnostics drops published diagnostics for uri once new text has
// been sent, so a read waits for the server's diagnostics of that text.
func (c *rpcClient) forgetDiagnostics(uri string) {
	c.mu.Lock()
	delete(c.diagnostics, uri)
	delete(c.diagVersion, uri)
	delete(c.diagAt, uri)
	c.mu.Unlock()
}

func (c *rpcClient) notifyContext(ctx context.Context, method string, params any) error {
	done := make(chan error, 1)
	go func() { done <- c.notify(method, params) }()
	select {
	case e := <-done:
		return e
	case <-ctx.Done():
		c.fail(ctx.Err())
		return ctx.Err()
	case <-c.done:
		return c.failure()
	}
}
