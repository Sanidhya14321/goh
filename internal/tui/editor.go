package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// editorDoneMsg carries the edited draft back after the editor exits.
type editorDoneMsg struct {
	text string
	err  error
	path string
}

// editorCommand resolves $VISUAL, then $EDITOR, splitting shell-style words so
// values such as "code --wait" work. It never invokes a shell.
func editorCommand(path string) (*exec.Cmd, error) {
	value := strings.TrimSpace(os.Getenv("VISUAL"))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if value == "" {
		return nil, errors.New("set $VISUAL or $EDITOR to edit the prompt in an external editor")
	}
	words, err := splitWords(value)
	if err != nil || len(words) == 0 {
		return nil, fmt.Errorf("cannot parse editor command %q", value)
	}
	return exec.Command(words[0], append(words[1:], path)...), nil
}

// normalizeEdited drops the single trailing newline editors add to a one-line
// prompt, keeping intentional multi-line endings.
func normalizeEdited(text string) string {
	for _, end := range []string{"\r\n", "\n"} {
		if body, ok := strings.CutSuffix(text, end); ok && !strings.ContainsAny(body, "\r\n") {
			return body
		}
	}
	return text
}

// openEditor suspends the UI, edits the draft in a private temporary file and
// restores the composer on return. The draft is kept if the editor fails.
func (m *model) openEditor() tea.Cmd {
	f, err := os.CreateTemp("", "goh-prompt-*.md")
	if err != nil {
		m.status = "Editor unavailable: " + err.Error()
		return nil
	}
	path := f.Name()
	_, err = f.WriteString(m.input)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	cmd, cmdErr := editorCommand(path)
	if err == nil {
		err = cmdErr
	}
	if err != nil {
		os.Remove(path)
		m.status = err.Error()
		return nil
	}
	if m.a != nil {
		cmd.Dir = m.a.Store.Project
	}
	return tea.ExecProcess(cmd, func(runErr error) tea.Msg {
		defer os.Remove(path)
		if runErr != nil {
			return editorDoneMsg{err: fmt.Errorf("editor exited with an error; draft unchanged: %w", runErr)}
		}
		b, readErr := os.ReadFile(path)
		return editorDoneMsg{text: string(b), err: readErr}
	})
}

// applyEditor replaces the draft with the edited text. Empty output keeps the
// original draft, matching the usual "quit without saving" expectation.
func (m *model) applyEditor(msg editorDoneMsg) {
	if msg.err != nil {
		m.status = msg.err.Error()
		return
	}
	text := normalizeEdited(msg.text)
	if strings.TrimSpace(text) == "" {
		m.status = "Editor returned an empty prompt; draft unchanged"
		return
	}
	m.input, m.inputCursor = "", 0
	m.insertInput(text)
	m.status = "Draft updated from editor"
}

// splitWords splits an editor setting into arguments, honouring single and
// double quotes and backslash escapes outside single quotes.
func splitWords(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord, quote, escaped := false, rune(0), false
	for _, r := range s {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\' && quote != '\'':
			escaped, inWord = true, true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 || escaped {
		return nil, errors.New("unterminated quote or escape")
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}
