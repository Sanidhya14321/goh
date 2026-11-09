package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/testutil"
)

func TestRuntimeLSPHelper(t *testing.T) {
	if os.Getenv("LSP_RUNTIME_HELPER") != "1" {
		return
	}
	testutil.RunLSP()
	os.Exit(0)
}
func runtimeLSP(t *testing.T, mode string) (*Agent, string) {
	t.Helper()
	a, root := testAgent(t)
	if e := os.WriteFile(filepath.Join(root, "a.lsptest"), []byte("a😀x\n"), 0600); e != nil {
		t.Fatal(e)
	}
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	a.Config.LSP = map[string]core.LSPConfig{"fixture": {Command: binary, Args: []string{"-test.run=^TestRuntimeLSPHelper$"}, Extensions: []string{".lsptest"}, Env: map[string]string{"LSP_RUNTIME_HELPER": "1", "LSP_FIXTURE_MODE": mode}}}
	return a, root
}
func TestLSPRuntimeDualApprovalDurabilityAndFailure(t *testing.T) {
	for _, mode := range []string{"no-tool-grant", "no-start-grant", "approved", "plan", "bad-initialize"} {
		t.Run(mode, func(t *testing.T) {
			fixtureMode := ""
			if mode == "bad-initialize" {
				fixtureMode = mode
			}
			a, _ := runtimeLSP(t, fixtureMode)
			if mode != "no-tool-grant" {
				a.Grants["lsp"] = true
			}
			if mode != "no-start-grant" {
				a.Grants["lsp_start:fixture"] = true
			}
			if mode == "plan" {
				a.Config.AgentMode = "plan"
			}
			turn := 0
			a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
				turn++
				if turn == 1 {
					return answer(toolCall("hover", "lsp", map[string]any{"operation": "hover", "filePath": "a.lsptest", "line": 1, "character": 3})), nil
				}
				if mode == "approved" {
					out := toolResult(t, h[len(h)-1])
					if out["operation"] != "hover" {
						t.Fatal(out)
					}
				}
				return answer(), nil
			})
			status, e := a.Run(context.Background(), "inspect hover")
			switch mode {
			case "no-tool-grant", "no-start-grant":
				if status != "needs_permission" || e == nil {
					t.Fatal(status, e)
				}
			case "bad-initialize":
				// The broken server is stopped and settled; the run continues
				// with the LSP error as a tool result and nothing pending.
				if status != "completed" || e != nil || len(a.Store.Pending()) != 0 || !strings.Contains(journalText(a), "integration_degraded") {
					t.Fatal(status, e, a.Store.Pending())
				}
			default:
				if status != "completed" || e != nil {
					t.Fatal(status, e)
				}
			}
			starts, results := 0, 0
			for _, r := range a.Store.Records() {
				if r.Type == "tool_start" {
					starts++
				}
				if r.Type == "tool_result" {
					results++
				}
			}
			if mode == "approved" && (starts != 2 || results != 2 || len(a.Store.Pending()) != 0) {
				t.Fatal("startup/query pairing", starts, results)
			}
			if mode == "plan" && starts > 0 {
				t.Fatal("plan started language server")
			}
			if mode != "bad-initialize" && len(a.Store.Pending()) > 0 {
				t.Fatal("unapproved/complete operation left uncertain")
			}
		})
	}
}
func TestLSPInterruptedOperationIsNotReplayed(t *testing.T) {
	a, _ := runtimeLSP(t, "hang")
	a.Grants["lsp"] = true
	a.Grants["lsp_start:fixture"] = true
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(toolCall("hover", "lsp", map[string]any{"operation": "hover", "filePath": "a.lsptest", "line": 1, "character": 1})), nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	status, e := a.Run(ctx, "cancel hover")
	if status != "cancelled" || e == nil || len(a.Store.Pending()) == 0 {
		t.Fatal(status, e, a.Store.Pending())
	}
	observed := false
	for _, r := range a.Store.Records() {
		observed = observed || r.Type == "tool_observation"
	}
	if !observed {
		t.Fatal("interruption not observed")
	}
	resumed, e := New(a.Config, a.Store)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = resumed.Run(context.Background(), "continue"); e == nil || !strings.Contains(e.Error(), "uncertain") {
		t.Fatal("uncertain LSP replay allowed", e)
	}
}

func TestLSPMissingExecutableDegradesWithoutPendingEffects(t *testing.T) {
	a, root := runtimeLSP(t, "")
	a.Config.LSP = map[string]core.LSPConfig{"fixture": {Command: filepath.Join(root, "no-such-lsp"), Extensions: []string{".lsptest"}}}
	a.Grants["lsp"], a.Grants["lsp_start:fixture"] = true, true
	turn := 0
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		if turn == 1 {
			return answer(toolCall("hover", "lsp", map[string]any{"operation": "hover", "filePath": "a.lsptest", "line": 1, "character": 1})), nil
		}
		if c := h[len(h)-1].Content; !strings.Contains(c, `\"status\":\"failed\"`) {
			t.Fatal("expected settled failure", c)
		}
		return answer(), nil
	})
	if status, e := a.Run(context.Background(), "hover"); status != "completed" || e != nil || len(a.Store.Pending()) != 0 {
		t.Fatal(status, e, a.Store.Pending())
	}
	// The next run must not be blocked by a start that never spawned.
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		return answer(), nil
	})
	if status, e := a.Run(context.Background(), "continue"); status != "completed" || e != nil {
		t.Fatal("follow-up blocked", status, e)
	}
}

func TestEditReportsErrorsFromRunningLanguageServer(t *testing.T) {
	a, _ := runtimeLSP(t, "")
	a.Grants["lsp"], a.Grants["lsp_start:fixture"], a.Grants["write_file"], a.Grants["create_file"] = true, true, true, true
	turn := 0
	var editResult string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		switch turn {
		case 1: // start the server through an approved query
			return answer(toolCall("hover", "lsp", map[string]any{"operation": "hover", "filePath": "a.lsptest", "line": 1, "character": 1})), nil
		case 2:
			return answer(toolCall("make", "create_file", map[string]any{"path": "b.lsptest", "content": "ok\nBROKEN here\n"})), nil
		}
		editResult = h[len(h)-1].Content
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "edit"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	if !strings.Contains(editResult, `lsp_errors`) || !strings.Contains(editResult, `2:5 Fixture error: BROKEN`) {
		t.Fatal("edit result lacks the language-server error", editResult)
	}
	if strings.Contains(editResult, "Fixture diagnostic") {
		t.Fatal("warnings must not be reported as edit errors", editResult)
	}
}

func TestEditDoesNotStartLanguageServer(t *testing.T) {
	a, _ := runtimeLSP(t, "")
	a.Grants["create_file"] = true
	a.Approve = func(_ context.Context, p core.Permission) (core.Decision, error) {
		if strings.HasPrefix(p.Tool, "lsp_start:") {
			t.Fatal("an edit must not request a language-server start")
		}
		return core.Decision{}, nil
	}
	turn := 0
	var editResult string
	a.Provider = modelFunc(func(_ context.Context, h []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
		turn++
		if turn == 1 {
			return answer(toolCall("make", "create_file", map[string]any{"path": "b.lsptest", "content": "BROKEN\n"})), nil
		}
		editResult = h[len(h)-1].Content
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "edit"); status != "completed" || err != nil {
		t.Fatal(status, err)
	}
	if strings.Contains(editResult, "lsp_errors") {
		t.Fatal("no server was running, so no errors should be attached", editResult)
	}
}
