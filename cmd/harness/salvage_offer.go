package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Sanidhya14321/goh/internal/session"
)

// interactive reports whether standard input is a terminal.
func interactive() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// offerSalvage asks before recovering a damaged session's readable records
// into a new session. The damaged journal is never changed; declining (or
// any answer but yes) keeps the original error.
func offerSalvage(in io.Reader, out io.Writer, dir, id string, damaged error) (*session.Store, error) {
	fmt.Fprintf(out, "%v\nSession %s is damaged. Recover its readable conversation into a new session? The damaged file is kept unchanged. [y/N] ", damaged, id)
	answer, _ := bufio.NewReader(io.LimitReader(in, 64)).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		return nil, fmt.Errorf("%w\nNothing was changed. Recover later with:\n  goh sessions salvage %s", damaged, id)
	}
	store, report, err := session.Salvage(dir, id)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(out, "Recovered %d records into session %s (the damage was at byte %d: %s).\n", report.KeptRecords, report.Session, report.DamagedAt, report.Problem)
	return store, nil
}
