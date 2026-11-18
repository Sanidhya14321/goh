package main

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tui"
)

// chooseAccess is /access: pick a level, then where it applies (this
// session, the project, or one folder inside it). It returns a notice for
// the next chat screen.
func chooseAccess(project, backdrop string) (string, error) {
	if r, err := filepath.EvalSymlinks(project); err == nil {
		project = r
	}
	current := tui.SessionAccess()
	if current == "" {
		current = core.AccessAsk
		if r, ok := core.AccessFor(core.SavedAccess(), project); ok {
			current = r.Level
		}
	}
	var levels []tui.Option
	for _, l := range core.AccessLevels {
		levels = append(levels, tui.Option{Value: string(l), Label: l.Label(), Description: l.Describe()})
	}
	choice, ok, err := tui.SelectOptionFrom("Access · how much goh does without asking", string(current), levels, backdrop, "",
		"  ↑/↓ select  Enter next  Esc back · deny and ask rules in settings always apply")
	if err != nil || !ok {
		return "", err
	}
	level := core.AccessLevel(choice)
	where := []tui.Option{
		{Value: "session", Label: "This session only", Description: "Until goh exits; Shift+Tab changes it"},
		{Value: "project", Label: "Always in " + displayPath(project), Description: "Saved for this folder and its subfolders"},
		{Value: "folder", Label: "Only in a folder inside it…", Description: "Choose a subfolder; the rest of the project keeps its level"},
	}
	scope, ok, err := tui.SelectOptionFrom(level.Label()+" · where", "session", where, backdrop, "", "")
	if err != nil || !ok {
		return "", err
	}
	switch scope {
	case "session":
		tui.SetSessionAccessLevel(level)
		return "Access: " + level.Label() + " for this session.", nil
	case "project":
		if err := core.SaveFolderAccess(project, level); err != nil {
			return "", err
		}
		tui.SetSessionAccessLevel("") // the saved level applies from now on
		return "Access: " + level.Label() + " saved for " + displayPath(project) + ".", nil
	}
	folder, ok, err := tui.ChooseWorkspace(tui.WorkspaceChooser{
		Start: project,
		Check: func(path string) error {
			if r, err := filepath.EvalSymlinks(path); err == nil {
				path = r
			}
			if rel, err := filepath.Rel(project, path); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return errors.New("choose a folder inside " + displayPath(project))
			}
			return nil
		},
		Trusted: func(string) bool { return true }, // choosing here grants access, not trust
		Kind:    core.ProjectKind,
	})
	if err != nil || !ok {
		return "", err
	}
	if err := core.SaveFolderAccess(folder, level); err != nil {
		return "", err
	}
	return "Access: " + level.Label() + " saved for " + displayPath(folder) + ".", nil
}
