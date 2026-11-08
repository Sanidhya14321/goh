package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(st)
	}
	fmt.Printf("Sessions: %d   Prompts: %d\n", st.Sessions, st.Prompts)
	if len(st.Models) == 0 {
		fmt.Println("No token usage recorded yet (recorded from 1.0.0-rc.60 when the provider reports it).")
	}
	for _, m := range st.Models {
		fmt.Printf("%-32s %6d replies  %12d in  %12d out  %12d total\n", m.Model, m.Replies, m.Prompt, m.Completion, m.Total)
	}
	if st.Damaged > 0 {
		fmt.Printf("%d damaged session(s) skipped; see goh sessions list\n", st.Damaged)
	}
	return nil
}
