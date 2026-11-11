package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/tui"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

// stats prints session and token totals from local journals; nothing is sent
// anywhere.
func stats(args []string) error {
	f := flag.NewFlagSet("stats", flag.ContinueOnError)
	defaultDir := configuredSessionDir()
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
	root, err := projectFilter(*project)
	if err != nil {
		return err
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
			st.Models[i].Estimate(p.Input, p.Output, p.CacheRead, p.CacheWrite)
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
		cache := ""
		if m.CacheRead > 0 || m.CacheWrite > 0 {
			cache = fmt.Sprintf("  (cache: %d read, %d written)", m.CacheRead, m.CacheWrite)
		}
		fmt.Fprintf(w, "%-32s %6d replies  %12d in  %12d out  %12d total%s%s\n", tui.Safe(m.Model), m.Replies, m.Prompt, m.Completion, m.Total, cost, cache)
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

// projectFilter resolves a --project argument the way sessions record their
// project (absolute, symlinks resolved); empty means no filter.
func projectFilter(dir string) (string, error) {
	if dir == "" {
		return "", nil
	}
	abs, err := filepath.Abs(dir)
	if err == nil {
		abs, err = filepath.EvalSymlinks(abs)
	}
	return abs, err
}
