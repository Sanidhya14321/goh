package runtime

import (
	"path/filepath"
	"strings"
)

// readOnlyPrograms only read (with the flags refused below). git and a few
// tools are allowed only with the listed subcommands.
var readOnlyPrograms = map[string]bool{
	"ls": true, "pwd": true, "cat": true, "head": true, "tail": true, "wc": true, "file": true, "stat": true,
	"which": true, "echo": true, "true": true, "date": true, "uname": true, "whoami": true, "tree": true,
	"du": true, "df": true, "sort": true, "cut": true, "tr": true, "grep": true, "egrep": true,
	"fgrep": true, "rg": true, "find": true, "diff": true, "cmp": true, "basename": true, "dirname": true,
	"realpath": true, "md5": true, "md5sum": true, "shasum": true, "sha256sum": true, "jq": true, "nl": true,
	"column": true, "test": true, "[": true, "git": true, "go": true, "command": true, "node": true,
	"python3": true, "python": true, "npm": true, "cargo": true, "rustc": true, "java": true,
}

var readOnlyGit = map[string]bool{"status": true, "diff": true, "log": true, "show": true, "rev-parse": true,
	"ls-files": true, "blame": true, "grep": true, "describe": true, "shortlog": true, "remote": true, "branch": true, "tag": true}

// refusedFlags write files or run other programs, whatever the program.
var refusedFlags = map[string]bool{"-exec": true, "-execdir": true, "-ok": true, "-okdir": true, "-delete": true,
	"-fprint": true, "-fprint0": true, "-fprintf": true, "-fls": true, "--output": true, "--write": true,
	"--in-place": true, "--output-directory": true, "--pre": true, "--pre-glob": true}

// refusedFor are flags that write only for these programs ("ls -d" lists a
// folder; "git branch -d" deletes a branch; "sort -o" writes a file).
var refusedFor = map[string]map[string]bool{
	"git":  {"-d": true, "-D": true, "-m": true, "-M": true, "-c": true, "-C": true, "-f": true, "--delete": true, "--move": true, "--copy": true, "--force": true, "--set-upstream-to": true, "--unset-upstream": true, "--edit-description": true},
	"sort": {"-o": true},
	"tree": {"-o": true},
	"date": {"-s": true, "--set": true},
	"find": {"-fprint": true},
}

// readOnlyCommand reports whether cmd only reads inside the project: every
// part of it (joined by ;, &&, || or |) is a reading program, nothing is
// redirected into a file, nothing is substituted, and every path stays in
// the project (no absolute path, ~ or ..). It errs on the side of asking.
func readOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || len(cmd) > 2000 || strings.ContainsAny(cmd, "`\n\r\\") || strings.Contains(cmd, "$(") || strings.Contains(cmd, "${") {
		return false
	}
	// Discarding output is fine; any other redirection could write a file.
	for _, ok := range []string{"2>/dev/null", "2> /dev/null", ">/dev/null", "> /dev/null", "2>&1", "&>/dev/null"} {
		cmd = strings.ReplaceAll(cmd, ok, " ")
	}
	if strings.ContainsAny(cmd, "<>") {
		return false
	}
	segments := splitShell(cmd)
	if segments == nil {
		return false
	}
	for _, words := range segments {
		if !readOnlySegment(words) {
			return false
		}
	}
	return true
}

// readOnlySegment checks one simple command.
func readOnlySegment(words []string) bool {
	if len(words) == 0 {
		return false
	}
	name := words[0]
	if strings.Contains(name, "/") || strings.Contains(name, "=") || !readOnlyPrograms[name] {
		return false
	}
	args := words[1:]
	switch name {
	case "git":
		if len(args) == 0 || !readOnlyGit[args[0]] {
			return false
		}
		if args[0] == "remote" && (len(args) != 2 || args[1] != "-v") && len(args) != 1 {
			return false
		}
		if (args[0] == "branch" || args[0] == "tag") && len(args) > 1 {
			for _, a := range args[1:] {
				if !strings.HasPrefix(a, "-") || refusedFor["git"][a] {
					return false // creating, renaming or deleting
				}
			}
		}
	case "go":
		if len(args) == 0 || (args[0] != "version" && args[0] != "env" && args[0] != "list" && args[0] != "doc") {
			return false
		}
		if args[0] == "env" && len(args) > 1 && strings.HasPrefix(args[1], "-w") {
			return false
		}
	case "node", "python3", "python", "rustc", "java", "cargo":
		return len(args) == 1 && (args[0] == "--version" || args[0] == "-v" || args[0] == "-V" || args[0] == "-version")
	case "npm":
		return len(args) >= 1 && (args[0] == "--version" || args[0] == "-v" || args[0] == "ls" || args[0] == "list" || args[0] == "view")
	case "command":
		return len(args) >= 2 && args[0] == "-v"
	case "file":
		for _, a := range args {
			if strings.HasPrefix(a, "-C") || strings.HasPrefix(a, "--compile") {
				return false // writes a compiled magic file
			}
		}
	}
	for _, a := range args {
		if refusedFlags[a] || refusedFor[name][a] || strings.HasPrefix(a, "--output=") || strings.HasPrefix(a, "-fprint") || (name == "sort" && strings.HasPrefix(a, "-o")) {
			return false
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		// "/x" and "C:x" count as outside on every system (filepath.IsAbs
		// alone misses "/x" on Windows).
		if filepath.IsAbs(a) || strings.HasPrefix(a, "/") || (len(a) >= 2 && a[1] == ':') || strings.HasPrefix(a, "~") || a == ".." || strings.HasPrefix(a, "../") || strings.Contains(a, "/../") || strings.HasSuffix(a, "/..") {
			return false
		}
	}
	return true
}

// splitShell splits a command into simple commands at ;, &&, || and |, and
// each into words, honouring single and double quotes. It returns nil for
// anything it does not understand (unbalanced quotes, background &).
func splitShell(cmd string) [][]string {
	var out [][]string
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
	flushCmd := func() bool {
		flushWord()
		if len(words) == 0 {
			return false
		}
		out = append(out, words)
		words = nil
		return true
	}
	r := []rune(cmd)
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				word.WriteRune(c)
			}
		case c == '\'' || c == '"':
			quote, inWord = c, true
		case c == ' ' || c == '\t':
			flushWord()
		case c == ';' || c == '|':
			if c == '|' && i+1 < len(r) && r[i+1] == '|' {
				i++
			}
			if !flushCmd() {
				return nil
			}
		case c == '&':
			if i+1 >= len(r) || r[i+1] != '&' {
				return nil // background job
			}
			i++
			if !flushCmd() {
				return nil
			}
		default:
			word.WriteRune(c)
			inWord = true
		}
	}
	if quote != 0 || !flushCmd() {
		return nil
	}
	return out
}
