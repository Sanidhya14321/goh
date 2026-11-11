// Package localapi exposes a single project's runtime over authenticated loopback HTTP.
package localapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/mcpclient"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
)

type Server struct {
	config  core.Config
	project string
	token   [sha256.Size]byte
	routes  *http.ServeMux

	mu     sync.Mutex
	active *run
}

type run struct {
	id     string
	cancel context.CancelFunc
	input  *pendingInput
}

type pendingInput struct {
	id       string
	kind     string
	question core.QuestionRequest
	reply    chan inputReply
}

type inputReply struct {
	Allow   *bool      `json:"allow,omitempty"`
	Answers [][]string `json:"answers,omitempty"`
	Dismiss bool       `json:"dismiss,omitempty"`
}

type inputFrame struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Data any    `json:"data"`
}

func New(config core.Config, project, token string) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if len(token) < 32 || len(token) > 512 || strings.ContainsAny(token, " \t\r\n") {
		return nil, errors.New("server bearer token must be 32 to 512 bytes without whitespace")
	}
	project, err := filepath.Abs(project)
	if err != nil {
		return nil, err
	}
	project, err = filepath.EvalSymlinks(project)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(project)
	if err != nil || !info.IsDir() {
		return nil, errors.New("server project must be an existing directory")
	}
	s := &Server{config: config, project: project, token: sha256.Sum256([]byte(token)), routes: http.NewServeMux()}
	s.routes.HandleFunc("GET /v1/sessions", s.list)
	s.routes.HandleFunc("POST /v1/sessions", s.create)
	s.routes.HandleFunc("POST /v1/sessions/{session}/run", s.prompt)
	s.routes.HandleFunc("POST /v1/runs/{run}/inputs/{input}", s.submit)
	s.routes.HandleFunc("POST /v1/runs/{run}/cancel", s.cancel)
	return s, nil
}

// IsLoopbackHost accepts only numeric loopback addresses, with an optional port.
func IsLoopbackHost(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback() && ip.Zone() == ""
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if origin := r.Header.Get("Origin"); !IsLoopbackHost(r.Host) || (origin != "" && origin != "http://"+r.Host) {
		problem(w, http.StatusForbidden, "invalid host or origin")
		return
	}
	value, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	digest := sha256.Sum256([]byte(value))
	if !ok || subtle.ConstantTimeCompare(s.token[:], digest[:]) != 1 {
		w.Header().Set("WWW-Authenticate", "Bearer")
		problem(w, http.StatusUnauthorized, "server bearer token required")
		return
	}
	s.routes.ServeHTTP(w, r)
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func problem(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}

func decode(w http.ResponseWriter, r *http.Request, dst any, max int64) bool {
	body := http.MaxBytesReader(w, r.Body, max)
	defer body.Close()
	d := json.NewDecoder(body)
	d.DisallowUnknownFields()
	var err error
	if err = d.Decode(dst); err == nil {
		err = d.Decode(new(any))
		if err == io.EOF {
			return true
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		problem(w, http.StatusRequestEntityTooLarge, "request body exceeds limit")
	} else {
		problem(w, http.StatusBadRequest, "expected one JSON object with supported fields")
	}
	return false
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if value := r.URL.Query().Get("limit"); value != "" {
		var err error
		limit, err = strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			problem(w, http.StatusBadRequest, "limit must be 1 to 100")
			return
		}
	}
	sessions, err := session.List(s.config.SessionDir)
	if err != nil {
		problem(w, http.StatusInternalServerError, "cannot list sessions")
		return
	}
	page := make([]session.Info, 0, limit)
	after := r.URL.Query().Get("after")
	next := ""
	for _, info := range sessions {
		if info.Project != s.project || info.ID <= after {
			continue
		}
		if len(page) == limit {
			next = page[len(page)-1].ID
			break
		}
		page = append(page, info)
	}
	respond(w, http.StatusOK, map[string]any{"sessions": page, "next_after": next})
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength != 0 {
		problem(w, http.StatusBadRequest, "session creation takes no body")
		return
	}
	store, err := session.New(s.config.SessionDir, s.project)
	if err != nil {
		problem(w, http.StatusInternalServerError, "cannot create session")
		return
	}
	info := session.Info{ID: store.ID, Project: store.Project}
	if err := store.Close(); err != nil {
		problem(w, http.StatusInternalServerError, "cannot close session journal")
		return
	}
	respond(w, http.StatusCreated, info)
}

func (s *Server) prompt(w http.ResponseWriter, r *http.Request) {
	var request *struct {
		Prompt               string `json:"prompt"`
		AcknowledgeUncertain bool   `json:"acknowledge_uncertain,omitempty"`
	}
	if !decode(w, r, &request, 512<<10) {
		return
	}
	if request == nil || strings.TrimSpace(request.Prompt) == "" || len(request.Prompt) > 64<<10 {
		problem(w, http.StatusBadRequest, "prompt must be nonempty and at most 64 KiB")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()
	current := &run{cancel: cancel}
	s.mu.Lock()
	if s.active != nil {
		s.mu.Unlock()
		problem(w, http.StatusConflict, "server already has an active run")
		return
	}
	s.active = current
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.active = nil
		s.mu.Unlock()
	}()

	store, err := session.OpenProject(s.config.SessionDir, r.PathValue("session"), s.project)
	if err != nil {
		if errors.Is(err, session.ErrBusy) {
			problem(w, http.StatusConflict, "session already has an active writer")
		} else if errors.Is(err, session.ErrDamaged) {
			problem(w, http.StatusConflict, "session journal is damaged; recover it with `goh sessions salvage "+r.PathValue("session")+"`")
		} else {
			problem(w, http.StatusNotFound, "session unavailable in this project")
		}
		return
	}
	defer store.Close()
	if pending := store.Pending(); len(pending) > 0 {
		if !request.AcknowledgeUncertain {
			respond(w, http.StatusConflict, map[string]any{"error": "inspect uncertain effects before acknowledging them", "pending": pending})
			return
		}
		if err := store.Acknowledge(); err != nil {
			problem(w, http.StatusInternalServerError, "cannot acknowledge uncertain effects")
			return
		}
	}

	a, err := agent.New(s.config, store)
	if err != nil {
		problem(w, http.StatusInternalServerError, "cannot initialize session runtime")
		return
	}
	defer a.Tools.StopJobs() // background jobs end with the request
	a.ExternalFactory = func() agent.External { return mcpclient.New(s.config.MCP, store.Project, a.Permission) }
	w.Header().Set("Content-Type", "application/x-ndjson")
	controller := http.NewResponseController(w)
	encoder := json.NewEncoder(w)
	write := func(frame any) {
		// A stalled reader holds at most one frame and cannot stall cleanup.
		if r.Context().Err() != nil {
			cancel()
			return
		}
		if err := controller.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			cancel()
			return
		}
		defer controller.SetWriteDeadline(time.Time{})
		if err := encoder.Encode(frame); err != nil {
			cancel()
			return
		}
		if err := controller.Flush(); err != nil {
			cancel()
		}
	}
	a.Emit = func(event core.Event) {
		if event.Sequence == 1 {
			s.mu.Lock()
			current.id = event.RunID
			s.mu.Unlock()
			w.Header().Set("X-Harness-Run-ID", event.RunID)
		}
		write(struct {
			Event core.Event `json:"event"`
		}{event})
	}
	input := func(ctx context.Context, kind string, data any, question core.QuestionRequest) (inputReply, error) {
		pending := &pendingInput{id: agent.ID(), kind: kind, question: question, reply: make(chan inputReply, 1)}
		s.mu.Lock()
		current.input = pending
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			current.input = nil
			s.mu.Unlock()
		}()
		write(struct {
			Input inputFrame `json:"input"`
		}{inputFrame{ID: pending.id, Kind: kind, Data: data}})
		wait, stop := context.WithTimeout(ctx, 2*time.Minute)
		defer stop()
		select {
		case <-wait.Done():
			return inputReply{}, wait.Err()
		case reply := <-pending.reply:
			return reply, wait.Err()
		}
	}
	a.Approve = func(ctx context.Context, permission core.Permission) (core.Decision, error) {
		reply, err := input(ctx, "permission", permission, core.QuestionRequest{})
		return core.Decision{Allow: err == nil && reply.Allow != nil && *reply.Allow}, err
	}
	a.Ask = func(ctx context.Context, question core.QuestionRequest) (core.QuestionReply, error) {
		reply, err := input(ctx, "question", question, question)
		if err == nil && reply.Dismiss {
			err = agent.ErrQuestionDismissed
		}
		return core.QuestionReply{Answers: reply.Answers}, err
	}
	_, _ = a.Run(ctx, request.Prompt)
}

func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	var reply *inputReply
	if !decode(w, r, &reply, 64<<10) {
		return
	}
	if reply == nil {
		problem(w, http.StatusBadRequest, "reply must be an object")
		return
	}
	s.mu.Lock()
	status, message := s.deliver(r.PathValue("run"), r.PathValue("input"), *reply)
	s.mu.Unlock()
	if status == http.StatusNoContent {
		w.WriteHeader(status)
	} else {
		problem(w, status, message)
	}
}

// deliver requires the server mutex. Each challenge accepts one valid response.
func (s *Server) deliver(runID, inputID string, reply inputReply) (int, string) {
	current := s.active
	if current == nil || current.id != runID || current.input == nil || current.input.id != inputID {
		return http.StatusNotFound, "input challenge is no longer pending"
	}
	pending := current.input
	switch pending.kind {
	case "permission":
		if reply.Allow == nil || reply.Answers != nil || reply.Dismiss {
			return http.StatusBadRequest, "permission reply requires allow"
		}
	case "question":
		if reply.Allow != nil || (reply.Dismiss && reply.Answers != nil) {
			return http.StatusBadRequest, "question reply requires answers or dismiss"
		}
		if !reply.Dismiss {
			if _, err := agent.ValidateQuestionReply(pending.question, core.QuestionReply{Answers: reply.Answers}); err != nil {
				return http.StatusBadRequest, err.Error()
			}
		}
	}
	current.input = nil
	pending.reply <- reply
	return http.StatusNoContent, ""
}

func (s *Server) cancel(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	current := s.active
	if current != nil && current.id == r.PathValue("run") {
		current.cancel()
		s.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
		return
	}
	s.mu.Unlock()
	problem(w, http.StatusNotFound, "run is no longer active")
}
