package tools

import (
	"path/filepath"
	"regexp"
	"strings"
)

// declRule finds one kind of declaration on a single line. Regexes are not
// parsers: they cover common declaration shapes and can miss or misread
// unusual ones, so every symbol carries its line and the file hash for the
// model to check with read_file.
type declRule struct {
	kind string
	re   *regexp.Regexp
	name int // capture group holding the name
}

var (
	pyRules = []declRule{
		{"class", regexp.MustCompile(`^\s*class\s+([A-Za-z_]\w*)`), 1},
		{"function", regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_]\w*)\s*\(`), 1},
		{"method", regexp.MustCompile(`^\s+(?:async\s+)?def\s+([A-Za-z_]\w*)\s*\(`), 1},
	}
	jsRules = []declRule{
		{"function", regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\*?\s+([A-Za-z_$][\w$]*)`), 1},
		{"class", regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][\w$]*)`), 1},
		{"type", regexp.MustCompile(`^\s*(?:export\s+)?(?:declare\s+)?(?:interface|type|enum)\s+([A-Za-z_$][\w$]*)`), 1},
		{"function", regexp.MustCompile(`^\s*(?:export\s+)?const\s+([A-Za-z_$][\w$]*)\s*(?::[^=]+)?=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*(?::[^=]+)?=>`), 1},
	}
	rustRules = []declRule{
		{"function", regexp.MustCompile(`^\s*(?:pub(?:\([\w:]+\))?\s+)?(?:const\s+)?(?:async\s+)?(?:unsafe\s+)?fn\s+([A-Za-z_]\w*)`), 1},
		{"type", regexp.MustCompile(`^\s*(?:pub(?:\([\w:]+\))?\s+)?(struct|enum|trait|type|mod)\s+([A-Za-z_]\w*)`), 2},
		{"impl", regexp.MustCompile(`^\s*impl(?:<[^>]*>)?\s+([A-Za-z_][\w:<>, ]*?)(?:\s+for\s+[\w:<>]+)?\s*\{`), 1},
	}
	javaRules = []declRule{
		{"type", regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|static|final|abstract|sealed|open|data|partial)\s+)*(?:class|interface|enum|record|struct|object)\s+([A-Za-z_]\w*)`), 1},
		{"method", regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|static|final|abstract|synchronized|override|async|virtual|suspend)\s+)+[\w<>\[\],.? ]+?\s+([A-Za-z_]\w*)\s*\([^;]*$`), 1},
		{"function", regexp.MustCompile(`^\s*(?:(?:public|private|internal|override|suspend|inline)\s+)*fun\s+(?:<[^>]*>\s*)?(?:[\w.]+\.)?([A-Za-z_]\w*)\s*\(`), 1},
	}
	cRules = []declRule{
		{"type", regexp.MustCompile(`^\s*(?:typedef\s+)?(?:struct|class|enum|union|namespace)\s+([A-Za-z_]\w*)\s*(?:[:{]|$)`), 1},
		{"function", regexp.MustCompile(`^[A-Za-z_][\w\s\*&:<>,]*?[\s\*&]([A-Za-z_][\w:~]*)\s*\([^;]*\)\s*(?:const\s*)?(?:\{.*)?$`), 1},
	}
	rubyRules = []declRule{
		{"type", regexp.MustCompile(`^\s*(?:class|module)\s+([A-Z][\w:]*)`), 1},
		{"method", regexp.MustCompile(`^\s*def\s+((?:self\.)?[\w]+[?!=]?)`), 1},
	}
)

var langRules = map[string][]declRule{
	".py": pyRules, ".pyi": pyRules,
	".js": jsRules, ".jsx": jsRules, ".mjs": jsRules, ".cjs": jsRules, ".ts": jsRules, ".tsx": jsRules, ".mts": jsRules,
	".rs":   rustRules,
	".java": javaRules, ".kt": javaRules, ".kts": javaRules, ".cs": javaRules, ".scala": javaRules,
	".c": cRules, ".h": cRules, ".cc": cRules, ".cpp": cRules, ".cxx": cRules, ".hpp": cRules, ".hh": cRules,
	".rb": rubyRules,
}

// mapSupported reports whether repo_map can list declarations in path.
func mapSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := langRules[ext]
	return ok || ext == ".go"
}

var cKeywords = map[string]bool{"if": true, "for": true, "while": true, "switch": true, "return": true, "sizeof": true, "else": true, "catch": true, "new": true, "delete": true}

// regexSymbols lists declarations in a non-Go file, one per matching line.
func regexSymbols(path string, data []byte) mapCacheEntry {
	digest := hash(data)
	out := mapCacheEntry{Digest: digest}
	ext := strings.ToLower(filepath.Ext(path))
	rules := langRules[ext]
	// Control-flow words only look like calls in C-like syntax.
	keywordGuard := ext != ".py" && ext != ".pyi" && ext != ".rs" && ext != ".rb"
	inBlockComment := false
	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if inBlockComment {
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "/*") && !strings.Contains(trimmed, "*/") {
			inBlockComment = true
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		for _, r := range rules {
			m := r.re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			name := strings.TrimSpace(m[r.name])
			if name == "" || keywordGuard && cKeywords[name] {
				continue
			}
			if len(out.Symbols) >= mapMaxSymbols {
				out.Partial = true
				return out
			}
			out.Symbols = append(out.Symbols, mapSymbol{Path: path, Line: i + 1, Kind: r.kind, Name: name, Signature: shortSignature(trimmed), SHA: digest})
			break
		}
	}
	return out
}
