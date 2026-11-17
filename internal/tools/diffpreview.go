package tools

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// diffContext is the number of unchanged lines shown around each change.
const diffContext = 3

// maxPreviewLine caps one previewed line, in bytes.
const maxPreviewLine = 2048

// diffPreview renders an edit for approval as a unified diff: only the
// changed lines with a little context, so a one-line edit to a long file
// shows one line, not the whole file twice. Output is bounded; a cut is
// marked so the reviewer knows the preview is incomplete.
func diffPreview(path string, before, after []byte, create bool) string {
	return boundedDiff(path, before, after, create, MaxInline-4096)
}

// boundedDiff is diffPreview within budget bytes.
func boundedDiff(path string, before, after []byte, create bool, budget int) string {
	var out strings.Builder
	header := "--- " + path + "\n+++ " + path + "\n"
	if len(header) > 2048 {
		header = header[:2048] + " [path preview truncated]\n"
	}
	out.WriteString(header)
	if create {
		out.WriteString("(new file)\n")
	}
	// Removed and added lines have separate budgets so a large old text
	// never hides the proposed new one; long lines are cut individually.
	side := map[byte]int{'-': budget * 2 / 5, '+': budget * 2 / 5}
	truncated := false
	write := func(line string) bool {
		if len(line) > maxPreviewLine {
			n := maxPreviewLine
			for n > 0 && !utf8.RuneStart(line[n]) {
				n--
			}
			line, truncated = line[:n]+" … [line truncated]", true
		}
		if kind := line[0]; kind == '-' || kind == '+' {
			if side[kind] < len(line)+1 {
				truncated = true
				return true // keep going: the other side may still fit
			}
			side[kind] -= len(line) + 1
		}
		if out.Len()+len(line)+1 > budget {
			truncated = true
			return false
		}
		out.WriteString(line)
		out.WriteByte('\n')
		return true
	}
	finish := func() string {
		if truncated {
			out.WriteString("[preview truncated; narrow the edit for complete review]\n")
		}
		return out.String()
	}
	a, aEOL := diffLines(string(before))
	b, bEOL := diffLines(string(after))
	ops := lineDiff(a, b)
	if len(ops) == 0 || allEqual(ops) {
		if aEOL != bEOL {
			write("(only the final newline changes)")
		} else {
			write("(no changes)")
		}
		return finish()
	}
	for _, h := range hunks(ops) {
		if !write(fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.aStart, h.aLen, h.bStart, h.bLen)) {
			return finish()
		}
		for _, o := range ops[h.from:h.to] {
			if !write(string(o.kind) + safeLine(o.text)) {
				return finish()
			}
		}
	}
	if aEOL != bEOL {
		if bEOL {
			write(`\ No newline at end of the old file`)
		} else {
			write(`\ No newline at end of the new file`)
		}
	}
	return finish()
}

type diffOp struct {
	kind byte // ' ', '-' or '+'
	text string
}

type hunk struct {
	from, to                   int // ops range
	aStart, aLen, bStart, bLen int
}

// diffLines splits text into lines and reports whether it ended with a
// newline; the final newline does not produce an empty last line.
func diffLines(text string) ([]string, bool) {
	if text == "" {
		return nil, true
	}
	eol := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	return lines, eol
}

func safeLine(s string) string {
	if !utf8.ValidString(s) {
		return strings.ToValidUTF8(s, "�")
	}
	return s
}

func allEqual(ops []diffOp) bool {
	for _, o := range ops {
		if o.kind != ' ' {
			return false
		}
	}
	return true
}

// lineDiff returns an edit script from a to b. Common leading and trailing
// lines are matched directly; the differing middle uses a longest common
// subsequence when small enough, else it is shown as replaced.
func lineDiff(a, b []string) []diffOp {
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	var ops []diffOp
	for _, l := range a[:pre] {
		ops = append(ops, diffOp{' ', l})
	}
	ops = append(ops, middleDiff(a[pre:len(a)-suf], b[pre:len(b)-suf])...)
	for _, l := range a[len(a)-suf:] {
		ops = append(ops, diffOp{' ', l})
	}
	return ops
}

func middleDiff(a, b []string) []diffOp {
	var ops []diffOp
	if len(a)*len(b) == 0 || len(a)*len(b) > 1<<20 {
		for _, l := range a {
			ops = append(ops, diffOp{'-', l})
		}
		for _, l := range b {
			ops = append(ops, diffOp{'+', l})
		}
		return ops
	}
	// lcs[i][j]: common subsequence length of a[i:] and b[j:].
	w := len(b) + 1
	lcs := make([]int32, (len(a)+1)*w)
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i*w+j] = lcs[(i+1)*w+j+1] + 1
			} else {
				lcs[i*w+j] = max(lcs[(i+1)*w+j], lcs[i*w+j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{' ', a[i]})
			i, j = i+1, j+1
		case lcs[(i+1)*w+j] >= lcs[i*w+j+1]:
			ops = append(ops, diffOp{'-', a[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < len(a); i++ {
		ops = append(ops, diffOp{'-', a[i]})
	}
	for ; j < len(b); j++ {
		ops = append(ops, diffOp{'+', b[j]})
	}
	return ops
}

// hunks groups changes with diffContext lines around them; changes closer
// than twice the context share one hunk.
func hunks(ops []diffOp) []hunk {
	var out []hunk
	aLine, bLine := 1, 1
	type pos struct{ a, b int }
	at := make([]pos, len(ops)+1)
	for i, o := range ops {
		at[i] = pos{aLine, bLine}
		if o.kind != '+' {
			aLine++
		}
		if o.kind != '-' {
			bLine++
		}
	}
	at[len(ops)] = pos{aLine, bLine}
	for i := 0; i < len(ops); {
		if ops[i].kind == ' ' {
			i++
			continue
		}
		from := max(0, i-diffContext)
		end := i
		for end < len(ops) {
			if ops[end].kind != ' ' {
				end++
				continue
			}
			run := end
			for run < len(ops) && ops[run].kind == ' ' {
				run++
			}
			if run < len(ops) && run-end <= 2*diffContext {
				end = run
				continue
			}
			break
		}
		to := min(len(ops), end+diffContext)
		h := hunk{from: from, to: to, aStart: at[from].a, bStart: at[from].b}
		for _, o := range ops[from:to] {
			if o.kind != '+' {
				h.aLen++
			}
			if o.kind != '-' {
				h.bLen++
			}
		}
		out = append(out, h)
		i = to
	}
	return out
}
