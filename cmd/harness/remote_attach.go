package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/localapi"
	"github.com/Sanidhya14321/goh/internal/remote"
	agent "github.com/Sanidhya14321/goh/internal/runtime"
	"github.com/Sanidhya14321/goh/internal/session"
	"github.com/Sanidhya14321/goh/internal/tui"
)

// runAttach implements `goh attach [ADDRESS]`: the terminal UI for a running
// `goh serve`. Prompts run on the server (its model, tools, grants and
// sandbox); approvals and questions are answered here. The UI reads the
// session journal the server writes, so it never takes the writer lock;
// operations that would write the session locally are refused.
func runAttach(args []string) error {
	f := flag.NewFlagSet("attach", flag.ContinueOnError)
	sessionID := f.String("session", "", "server session to open")
	continueLast := f.Bool("continue", false, "open the server project's most recent session")
	ack := f.Bool("acknowledge-uncertain", false, "let the first prompt continue past interrupted effects, after inspecting the project")
	tokenEnv := f.String("token-env", "HARNESS_SERVER_TOKEN", "environment variable holding the server token")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	address := "http://127.0.0.1:4096"
	switch f.NArg() {
	case 0:
	case 1:
		address = f.Arg(0)
	default:
		return errors.New("usage: goh attach [ADDRESS] [--session ID | --continue] [--token-env NAME]")
	}
	if *continueLast && *sessionID != "" {
		return errors.New("use either --continue or --session, not both")
	}
	token := os.Getenv(*tokenEnv)
	// Tools never run here, but keep the token out of any child process.
	_ = os.Unsetenv(*tokenEnv)
	client, err := remote.New(address, token)
	if err != nil {
		return err
	}
	ctx := context.Background()
	info, err := client.Info(ctx)
	if err != nil {
		return err
	}
	id := *sessionID
	if *continueLast {
		list, err := client.Sessions(ctx)
		if err != nil {
			return err
		}
		if latest := newestSession(list); latest != "" {
			id = latest
		}
	}
	if id == "" {
		if id, err = client.Create(ctx); err != nil {
			return err
		}
	}
	acknowledge := *ack
	notice, draft := fmt.Sprintf("Attached to %s (project %s).", address, info.Project), ""
	for {
		a, err := attachedAgent(info, id)
		if err != nil {
			return err
		}
		a.Remote = func(ctx context.Context, prompt string) (string, error) {
			// Handlers are read at call time: the UI installs its own.
			status, err := client.Run(ctx, id, prompt, acknowledge, remote.Handlers{Emit: a.Emit, Approve: a.Approve, Ask: a.Ask})
			if errors.Is(err, remote.ErrUncertain) {
				return status, err
			}
			acknowledge = false // applies to the first prompt only
			reloadAttached(a, info, id)
			return status, err
		}
		tui.Version = version
		err = tui.Run(a, notice, draft)
		var nav *tui.ChatNavigation
		if !errors.As(err, &nav) {
			return err
		}
		notice, draft = "", nav.Draft
		switch nav.Action {
		case "new":
			if id, err = client.Create(ctx); err != nil {
				notice = err.Error()
			}
		case "sessions":
			list, err := client.Sessions(ctx)
			if err != nil {
				notice = err.Error()
				break
			}
			selected, chosen, err := tui.SelectSession(list, info.SessionDir, id, info.Project)
			if err != nil {
				notice = err.Error()
			} else if chosen && selected != "" {
				id = selected
			}
		default:
			notice = fmt.Sprintf("/%s is not available while attached; the server's configuration and history are managed on the server.", nav.Action)
		}
	}
}

// attachedAgent builds an agent over a read-only view of the server's
// session, for display; its runs are delegated to the server.
func attachedAgent(info localapi.Info, id string) (*agent.Agent, error) {
	store, err := session.OpenReadOnly(info.SessionDir, id, info.Project)
	if err != nil {
		return nil, fmt.Errorf("cannot read the server's session %s: %w", id, err)
	}
	c := core.Defaults()
	c.SessionDir = info.SessionDir
	c.Model, c.Agent = info.Model, info.Agent
	if c.Model == "" {
		c.Model = "server"
	}
	return agent.New(c, store)
}

// reloadAttached refreshes history and tasks after a run the server wrote.
func reloadAttached(a *agent.Agent, info localapi.Info, id string) {
	fresh, err := attachedAgent(info, id)
	if err != nil {
		return
	}
	a.Store, a.History, a.Todos = fresh.Store, fresh.History, fresh.Todos
}

// newestSession picks the latest session; IDs start with a sortable UTC
// timestamp.
func newestSession(list []session.Info) string {
	newest := ""
	for _, s := range list {
		newest = max(newest, s.ID)
	}
	return newest
}
