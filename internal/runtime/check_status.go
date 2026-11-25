package runtime

import "strings"

// checkStatusAttributable reports whether a shell command's exit status can
// stand for the recognized checks it runs: a failing check must make the
// whole command fail. That holds for a simple command, for checks joined by
// && (a failure ends the list with its status), and for a check that is the
// last command after ; or as the last stage of a pipe. It does not hold when
// a later command runs after the check (; or a newline), the check feeds a
// pipe, || handles a failure or skips a check, or the command uses syntax
// this reader does not follow (subshells, groups, substitutions, control
// keywords, negation, background jobs). Those are treated as advisory.
func checkStatusAttributable(command string) bool {
	segs, ops, ok := errexitList(command)
	if !ok || len(segs) == 0 {
		return false
	}
	for _, op := range ops {
		if op == "||" {
			return false
		}
	}
	for i, words := range segs {
		if len(words) > 0 && shellKeywords[words[0]] {
			return false
		}
		program := words[:0:0]
		for _, w := range words {
			if !strings.ContainsAny(w, "<>") {
				program = append(program, w) // classify without redirections (2>&1)
			}
		}
		if checkKind(strings.Join(program, " ")) == "" {
			continue
		}
		for _, op := range ops[i:] {
			if op != "&&" {
				return false // a later command's status replaces the check's
			}
		}
	}
	return true
}

// shellKeywords change how a status flows; their lists are not analysed.
var shellKeywords = map[string]bool{"!": true, "if": true, "then": true, "else": true, "elif": true, "fi": true,
	"while": true, "until": true, "for": true, "do": true, "done": true, "case": true, "esac": true,
	"trap": true, "exit": true, "return": true, "set": true, "function": true}

// checksAfterChanges reports whether a command that changed files ends with
// a recognized check that ran after every change: commands joined only by
// &&, the last one a check and an earlier one not (a formatter, a code
// generator, an edit). A failure anywhere ends the list with its status, so
// exit 0 means the check passed on the changed files. A check alone that
// changed files still does not count: its own writes are unchecked.
func checksAfterChanges(command string) bool {
	segs, ops, ok := errexitList(command)
	if !ok || len(segs) < 2 {
		return false
	}
	for _, op := range ops {
		if op != "&&" {
			return false
		}
	}
	program := func(words []string) string {
		kept := words[:0:0]
		for _, w := range words {
			if !strings.ContainsAny(w, "<>") {
				kept = append(kept, w)
			}
		}
		return strings.Join(kept, " ")
	}
	if checkKind(program(segs[len(segs)-1])) == "" {
		return false
	}
	for _, words := range segs[:len(segs)-1] {
		if p := program(words); len(programs(p)) > 0 && checkKind(p) == "" {
			return true
		}
	}
	return false
}

// shellList splits a command into simple commands and the operators between
// them (";", "&&", "||", "|", or "\n"). ok is false for syntax it does not
// follow: unbalanced quotes, an unquoted ( ) { } or backquote, $( ), a
// background &, or an empty command.
func shellList(cmd string) (segs [][]string, ops []string, ok bool) {
	cmd, ok = stripHeredocs(cmd)
	if !ok {
		return nil, nil, false
	}
	var words []string
	var word strings.Builder
	inWord := false
	quote := rune(0)
	flushWord := func() {
		if inWord {
			words = append(words, word.String())
			word.Reset()
			inWord = false
		}
	}
	flushCmd := func(op string) bool {
		flushWord()
		if len(words) == 0 {
			return false
		}
		segs = append(segs, words)
		words = nil
		if op != "" {
			ops = append(ops, op)
		}
		return true
	}
	r := []rune(strings.TrimSpace(cmd))
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case quote == '\'':
			if c == quote {
				quote = 0
			} else {
				word.WriteRune(c)
			}
		case quote == '"':
			switch {
			case c == quote:
				quote = 0
			case c == '`' || c == '$' && i+1 < len(r) && r[i+1] == '(':
				return nil, nil, false // substitution inside double quotes
			default:
				word.WriteRune(c)
			}
		case c == '\'' || c == '"':
			quote, inWord = c, true
		case c == '\\':
			if i+1 < len(r) && r[i+1] == '\n' {
				i++ // line continuation
				continue
			}
			if i+1 < len(r) {
				i++
				word.WriteRune(r[i])
				inWord = true
			}
		case c == ' ' || c == '\t':
			flushWord()
		case c == '\n' && !inWord && len(words) == 0:
			// A blank line, or a line ending in && or |: not a command.
			continue
		case c == '\n' || c == ';':
			op := string(c)
			if !flushCmd(op) {
				return nil, nil, false
			}
		case c == '|':
			op := "|"
			if i+1 < len(r) && r[i+1] == '|' {
				op, i = "||", i+1
			} else if i+1 < len(r) && r[i+1] == '&' {
				i++ // |& pipes stderr too
			}
			if !flushCmd(op) {
				return nil, nil, false
			}
		case c == '&':
			if i+1 < len(r) && r[i+1] == '&' {
				i++
				if !flushCmd("&&") {
					return nil, nil, false
				}
				continue
			}
			if i > 0 && (r[i-1] == '>' || r[i-1] == '<') || i+1 < len(r) && r[i+1] == '>' {
				word.WriteRune(c) // 2>&1, &>file: a redirection
				inWord = true
				continue
			}
			return nil, nil, false // background job
		case c == '(' || c == ')' || c == '{' || c == '}' || c == '`':
			return nil, nil, false
		case c == '$' && i+1 < len(r) && r[i+1] == '(':
			return nil, nil, false
		default:
			word.WriteRune(c)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, nil, false
	}
	flushWord()
	if len(words) == 0 && len(ops) > 0 && (ops[len(ops)-1] == ";" || ops[len(ops)-1] == "\n") {
		ops = ops[:len(ops)-1] // a trailing terminator ends the list
		return segs, ops, true
	}
	if !flushCmd("") {
		return nil, nil, false
	}
	return segs, ops, true
}

// errexitList is shellList for a script that starts with set -e (set -eu,
// set -euo pipefail, set -o errexit): any failing command then ends it with
// its status, so commands after ; or a newline are joined as by &&. The set
// command itself is dropped. Without set -e it is shellList.
func errexitList(command string) (segs [][]string, ops []string, ok bool) {
	segs, ops, ok = shellList(command)
	if !ok || len(segs) < 2 || !setsErrexit(segs[0]) || ops[0] == "|" || ops[0] == "||" {
		return segs, ops, ok
	}
	segs, ops = segs[1:], append([]string(nil), ops[1:]...)
	// set -e does not stop at a failure inside an && list (only at its last
	// command): check && x followed by more lines can fail and still exit 0.
	// So && may appear only after the last line break.
	last := -1
	for i, op := range ops {
		if op == ";" || op == "\n" {
			last = i
		}
	}
	for i := 0; i < last; i++ {
		if ops[i] == "&&" {
			return shellList(command)
		}
	}
	for i, op := range ops {
		if op == ";" || op == "\n" {
			ops[i] = "&&"
		}
	}
	return segs, ops, true
}

// setsErrexit reports whether words is a set command that turns on errexit.
func setsErrexit(words []string) bool {
	if len(words) < 2 || words[0] != "set" {
		return false
	}
	for i, w := range words[1:] {
		if w == "-o" && i+2 < len(words) && words[i+2] == "errexit" {
			return true
		}
		if strings.HasPrefix(w, "-") && !strings.HasPrefix(w, "--") && strings.Contains(w, "e") {
			return true
		}
	}
	return false
}

// stripHeredocs removes here-document bodies (cmd <<'PY' ... PY): they are
// a command's input, not commands. The <<WORD redirection itself stays. A
// body that reaches the end without its delimiter ends there, as in bash.
// ok is false when a quote is left open.
func stripHeredocs(cmd string) (string, bool) {
	if !strings.Contains(cmd, "<<") {
		return cmd, true
	}
	r := []rune(cmd)
	var out strings.Builder
	type pending struct {
		word string
		tabs bool
	}
	var docs []pending
	quote := rune(0)
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else if c == '\\' && quote == '"' && i+1 < len(r) {
				out.WriteRune(c)
				i++
				c = r[i]
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '\\' && i+1 < len(r):
			out.WriteRune(c)
			i++
			c = r[i]
		case c == '<' && i+1 < len(r) && r[i+1] == '<' && (i+2 >= len(r) || r[i+2] != '<') && (i == 0 || r[i-1] != '<'):
			j := i + 2
			tabs := false
			if j < len(r) && r[j] == '-' {
				tabs, j = true, j+1
			}
			for j < len(r) && (r[j] == ' ' || r[j] == '\t') {
				j++
			}
			var word strings.Builder
			for j < len(r) && !strings.ContainsRune(" \t\n;&|<>()", r[j]) {
				if r[j] == '\'' || r[j] == '"' {
					q := r[j]
					j++
					for j < len(r) && r[j] != q {
						word.WriteRune(r[j])
						j++
					}
					if j >= len(r) {
						return cmd, false
					}
				} else if r[j] != '\\' {
					word.WriteRune(r[j])
				}
				j++
			}
			if word.Len() == 0 {
				return cmd, false
			}
			docs = append(docs, pending{word.String(), tabs})
			out.WriteString("<<" + word.String())
			i = j - 1
			continue
		case c == '\n' && len(docs) > 0:
			out.WriteRune(c)
			pos := i + 1
			for _, d := range docs {
				for pos < len(r) {
					end := pos
					for end < len(r) && r[end] != '\n' {
						end++
					}
					line := string(r[pos:end])
					if d.tabs {
						line = strings.TrimLeft(line, "\t")
					}
					pos = end + 1
					if line == d.word {
						break
					}
				}
			}
			docs = nil
			i = pos - 1
			continue
		}
		out.WriteRune(c)
	}
	if quote != 0 {
		return cmd, false
	}
	return out.String(), true
}
