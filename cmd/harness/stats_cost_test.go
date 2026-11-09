package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

func TestCostEstimates(t *testing.T) {
	st := session.Stats{Models: []session.ModelUsage{
		{Model: "priced", Replies: 2, Prompt: 2_000_000, Completion: 500_000},
		{Model: "free", Replies: 1, Prompt: 10},
	}}
	applyPrices(&st, map[string]core.ModelPrice{"priced": {Input: 1.5, Output: 6}})
	if st.Models[0].CostUSD == nil || *st.Models[0].CostUSD != 6 || st.Models[1].CostUSD != nil {
		t.Fatalf("%+v", st.Models)
	}
	var b bytes.Buffer
	writeModelRows(&b, st.Models)
	if !strings.Contains(b.String(), "~$6.0000") || !strings.Contains(b.String(), "Estimated cost: ~$6.0000") {
		t.Fatal(b.String())
	}
	b.Reset()
	writeModelRows(&b, []session.ModelUsage{{Model: "x"}})
	if !strings.Contains(b.String(), `"prices"`) {
		t.Fatal("no hint for unpriced usage")
	}
}
