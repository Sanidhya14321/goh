package main

import (
	"fmt"
	"io"
	"sort"

	"example.com/independent-harness/internal/session"
	"example.com/independent-harness/internal/tui"
)

// writeSessionTable prints sessions newest first for people; the JSON form
// stays the default for scripts.
func writeSessionTable(w io.Writer, dir string, infos []session.Info) error {
	rows := make([]session.PreviewInfo, 0, len(infos))
	for _, info := range infos {
		p, err := session.Preview(dir, info.ID)
		if err != nil {
			p = session.PreviewInfo{Info: info, Title: "(unreadable: " + err.Error() + ")"}
		}
		if meta, err := session.ReadMeta(dir, info.ID); err == nil && meta.Title != "" {
			p.Title = meta.Title
		}
		rows = append(rows, p)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Modified.After(rows[j].Modified) })
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "No saved sessions.")
		return err
	}
	for _, r := range rows {
		when := "-"
		if !r.Modified.IsZero() {
			when = r.Modified.Local().Format("2006-01-02 15:04")
		}
		title := []rune(tui.Safe(r.Title))
		if len(title) > 50 {
			title = append(title[:49], '…')
		}
		if _, err := fmt.Fprintf(w, "%s  %s  %-50s  %s\n", tui.Safe(r.ID), when, string(title), tui.Safe(r.Project)); err != nil {
			return err
		}
	}
	return nil
}
