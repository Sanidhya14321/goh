package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

// TestParentChildCancellationUnderLoad cancels several concurrent parents
// while each child is running a long shell command. Every run must return
// promptly, every child shell process must be gone, and the interrupted effect
// must stay pending so it is never replayed automatically.
func TestParentChildCancellationUnderLoad(t *testing.T) {
	const parents = 6
	var wg sync.WaitGroup
	errs := make(chan error, parents)
	for i := 0; i < parents; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := cancelOneParent(t, i); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func cancelOneParent(t *testing.T, i int) error {
	a, root := testAgent(t)
	a.Grants["task"], a.Grants["child:shell"] = true, true
	pidFile := filepath.Join(root, "child.pid")
	shell := toolCall("sleep", "shell", map[string]any{"command": sleepWritingPID, "timeout_seconds": 60})
	a.ChildModelFactory = func(core.Config, *session.Store) Model {
		turn := 0
		return modelFunc(func(ctx context.Context, _ []core.Message, _ []core.Tool, _ func(string)) (core.Reply, error) {
			turn++
			if turn == 1 {
				return answer(shell), nil
			}
			return answer(), nil
		})
	}
	parentTurn := 0
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		parentTurn++
		if parentTurn == 1 {
			return answer(toolCall("task", "task", map[string]any{"description": "sleep", "prompt": "run the sleep", "subagent_type": "general"})), nil
		}
		return answer(), nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		status string
		err    error
	}
	done := make(chan result, 1)
	go func() { s, e := a.Run(ctx, "start child "+strconv.Itoa(i)); done <- result{s, e} }()

	// Cancel once the child's shell process exists.
	// Shell start-up is slow on loaded Windows runners; this wait is not
	// what the test measures, so it is generous. A run that ends early is
	// reported with its outcome instead of as a timeout.
	var pid int
	for deadline := time.Now().Add(30 * time.Second); pid == 0 && time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if b, err := os.ReadFile(pidFile); err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		}
		select {
		case r := <-done:
			return fmt.Errorf("run ended before the child shell started: status %q, error %v", r.status, r.err)
		default:
		}
	}
	if pid == 0 {
		return errors.New("child shell never started within 30s")
	}
	started := time.Now()
	cancel()
	var r result
	select {
	case r = <-done:
	case <-time.After(10 * time.Second):
		return errors.New("parent did not return within 10s of cancellation")
	}
	if took := time.Since(started); took > 6*time.Second {
		return errors.New("cancellation took " + took.String())
	}
	if r.status == "completed" {
		return errors.New("cancelled run reported completed")
	}
	// The child's shell process (and its group) must be gone.
	gone := false
	for deadline := time.Now().Add(5 * time.Second); !gone && time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		gone = !processAlive(pid)
	}
	if !gone {
		killProcess(pid)
		return errors.New("child shell process survived cancellation")
	}
	// The interrupted task is an uncertain effect: the next run must refuse
	// rather than replay it.
	if len(a.Store.Pending()) == 0 {
		return errors.New("interrupted child effect was not left pending")
	}
	a.Provider = modelFunc(func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		t.Error("model called while uncertain effects are unresolved")
		return answer(), nil
	})
	if status, err := a.Run(context.Background(), "continue"); status != "failed" || err == nil || !strings.Contains(err.Error(), "acknowledge") {
		b, _ := json.Marshal(a.Store.Pending())
		return errors.New("follow-up run did not require acknowledgement: " + status + " " + string(b))
	}
	return nil
}
