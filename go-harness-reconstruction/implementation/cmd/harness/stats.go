package main

import (
	"encoding/json"
	"errors"
	"example.com/independent-harness/internal/tui"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/session"
)

// stats prints session and token totals from local journals; nothing is sent
// anywhere.
func stats(args []string) error {
	f := flag.NewFlagSet("stats", flag.ContinueOnError)
	defaultDir := core.Defaults().SessionDir
	if d, ok := os.LookupEnv("HARNESS_SESSION_DIR"); ok {
		defaultDir = d
	}
	dir := f.String("session-dir", defaultDir, "private session storage")
	days := f.Int("days", 0, "only sessions written in the last N days (0 = all)")
	project := f.String("project", "", "only sessions for this project directory")
	asJSON := f.Bool("json", false, "print JSON")
	configPath := f.String("config", "", "explicit user JSON configuration (for prices)")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 || *days < 0 {
		return errors.New("usage: goh stats [--days N] [--project DIR] [--json]")
	}
	var since time.Time
	if *days > 0 {
		since = time.Now().AddDate(0, 0, -*days)
	}
	root := ""
	if *project != "" {
		abs, err := filepath.Abs(*project)
		if err == nil {
			abs, err = filepath.EvalSymlinks(abs)
		}
		if err != nil {
			return err
		}
		root = abs
	}
	st, err := session.CollectStats(*dir, root, since)
	if err != nil {
		return err
	}
	if c, err := core.LoadConfig(*configPath); err == nil {
		applyPrices(&st, c.Prices)
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(st)
	}
	fmt.Printf("Sessions: %d   Prompts: %d\n", st.Sessions, st.Prompts)
	if len(st.Models) == 0 {
		fmt.Println("No token usage recorded yet (recorded from 1.0.0-rc.60 when the provider reports it).")
	}
	writeModelRows(os.Stdout, st.Models)
	writeToolRows(os.Stdout, st.Tools)
	if st.Damaged > 0 {
		fmt.Printf("%d damaged session(s) skipped; see goh sessions list\n", st.Damaged)
	}
	return nil
}

// applyPrices estimates cost for models with a configured price.
func applyPrices(st *session.Stats, prices map[string]core.ModelPrice) {
	for i := range st.Models {
		if p, ok := prices[st.Models[i].Model]; ok {
			st.Models[i].Estimate(p.Input, p.Output)
		}
	}
}

func writeModelRows(w io.Writer, models []session.ModelUsage) {
	var total float64
	priced := false
	for _, m := range models {
		cost := ""
		if m.CostUSD != nil {
			cost = fmt.Sprintf("  ~$%.4f", *m.CostUSD)
			total += *m.CostUSD
			priced = true
		}
		fmt.Fprintf(w, "%-32s %6d replies  %12d in  %12d out  %12d total%s\n", tui.Safe(m.Model), m.Replies, m.Prompt, m.Completion, m.Total, cost)
	}
	if priced {
		fmt.Fprintf(w, "Estimated cost: ~$%.4f (from configured prices; unpriced models excluded)\n", total)
	} else if len(models) > 0 {
		fmt.Fprintln(w, `Add "prices": {"<model>": {"input": USD_per_Mtok, "output": USD_per_Mtok}} to the config for cost estimates.`)
	}
}

func writeToolRows(w io.Writer, tools []session.ToolUsage) {
	if len(tools) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%-28s %8s %8s\n", "Tool", "Calls", "Errors")
	for i, t := range tools {
		if i == 15 {
			fmt.Fprintf(w, "... %d more (see --json)\n", len(tools)-15)
			break
		}
		fmt.Fprintf(w, "%-28s %8d %8d\n", tui.Safe(t.Name), t.Calls, t.Errors)
	}
}
