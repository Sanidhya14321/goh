package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
)

// costModel replies with tool calls (or a final answer after stop turns)
// and reports 10k prompt tokens, 2k of them cached, and 1k output each.
func costModel(stop int, served string) modelFunc {
	turn := 0
	return func(context.Context, []core.Message, []core.Tool, func(string)) (core.Reply, error) {
		turn++
		r := answer(toolCall(ID(), "glob", map[string]any{"pattern": strings.Repeat("*", turn)}))
		if turn > stop {
			r = answer()
		}
		r.Model = served
		r.Usage = &core.Usage{Prompt: 10_000, CacheRead: 2_000, Completion: 1_000, Total: 11_000}
		return r, nil
	}
}

func TestPriceCost(t *testing.T) {
	p := core.ModelPrice{Input: 2, Output: 10, CacheRead: 0.5}
	if got := p.Cost(1_000_000, 100_000, 200_000, 0); got < 2.699 || got > 2.701 { // 0.8*2 + 0.2*0.5 + 0.1*10
		t.Fatal(got)
	}
	c := core.Config{Prices: map[string]core.ModelPrice{"gpt-x": {Input: 1}, "gpt-x-mini": {Input: 2}}}
	for model, want := range map[string]float64{"gpt-x": 1, "gpt-x-2026-01-01": 1, "gpt-x-mini-2026": 2} {
		if p, ok := c.PriceFor(model); !ok || p.Input != want {
			t.Errorf("%s: %v %v", model, p, ok)
		}
	}
	if _, ok := c.PriceFor("gpt-xl"); ok {
		t.Fatal("prefix without a dash matched")
	}
}

func TestMaxRunCostStopsTheRun(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.Prices = map[string]core.ModelPrice{a.Config.Model: {Input: 200, Output: 1000, CacheRead: 50}}
	a.Config.MaxRunCost = 6 // $2.70 per request: stops before the fourth ($8.10 spent)
	a.Provider = costModel(10, a.Config.Model)
	status, err := a.Run(context.Background(), "go")
	if status != "budget_exhausted" || !errors.Is(err, errRunCost) || !strings.Contains(err.Error(), "used of $6.00") {
		t.Fatal(status, err)
	}
	if a.runCost < 6 || a.runCost > 9 {
		t.Fatal("cost", a.runCost)
	}
}

func TestMaxRunCostNeedsPrices(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.MaxRunCost = 5
	a.Provider = costModel(0, a.Config.Model)
	if status, err := a.Run(context.Background(), "go"); status != "budget_exhausted" || err == nil || !strings.Contains(err.Error(), "needs a price") {
		t.Fatal("unpriced main model", status, err)
	}
	// A request served by a model without a price stops the run after it.
	b, _ := testAgent(t)
	b.Config.Prices = map[string]core.ModelPrice{b.Config.Model: {Input: 1}}
	b.Config.MaxRunCost = 100
	b.Provider = costModel(5, "other-model")
	if status, err := b.Run(context.Background(), "go"); status != "budget_exhausted" || err == nil || !strings.Contains(err.Error(), "other-model") {
		t.Fatal("unpriced served model", status, err)
	}
}

func TestCostBudgetNote(t *testing.T) {
	a, _ := testAgent(t)
	a.Config.Prices = map[string]core.ModelPrice{a.Config.Model: {Input: 2, Output: 10, CacheRead: 0.5}}
	a.Config.MaxRunCost = 10
	a.runCost = 5.5
	a.resetBudget(a.budget.started)
	if note := a.budgetNote(context.Background()); !strings.Contains(note, "$4.50 of $10.00 left") {
		t.Fatal(note)
	}
}
