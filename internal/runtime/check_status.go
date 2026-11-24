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
	segs, ops, ok := shellList(command)
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
	segs, ops, ok := shellList(command)
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
