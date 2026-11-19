package tools

import (
	"context"
	"os"
	"regexp"
	"sort"
)

// relatedRef is one file that references a matched symbol: a one-hop
// expansion of repo_map's lexical hits (the retrieval-expansion
// experiment). Provenance says which symbol, defined where, led to it.
type relatedRef struct {
	Path       string `json:"path"`
	Line       int    `json:"line"`
	Symbol     string `json:"symbol"`
	DefinedIn  string `json:"defined_in"`
	Provenance string `json:"provenance"`
}

// refEntry is one file's identifier occurrences, cached by content digest
// so an edited file is re-read and an unchanged one is not.
type refEntry struct {
	digest string
	lines  map[string]int // identifier -> first line
}

const (
	relatedSymbols = 5  // matched symbols expanded
	relatedFiles   = 10 // files returned in all
)

var identifierRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

func (e *Engine) identifiers(path string, data []byte) refEntry {
	digest := hash(data)
	e.refMu.Lock()
	defer e.refMu.Unlock()
	if c, ok := e.refCache[path]; ok && c.digest == digest {
		return c
	}
	entry := refEntry{digest: digest, lines: map[string]int{}}
	line := 1
	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == '\n' {
			for _, id := range identifierRe.FindAll(data[start:i], -1) {
				if _, seen := entry.lines[string(id)]; !seen {
					entry.lines[string(id)] = line
				}
			}
			line++
			start = i + 1
		}
	}
	if e.refCache == nil {
		e.refCache = map[string]refEntry{}
	}
	if len(e.refCache) >= mapCacheEntries {
		e.refCache = map[string]refEntry{} // bounded: start over
	}
	e.refCache[path] = entry
	return entry
}

// relatedFiles finds, among the candidate files (already filtered by ignore
// and deny rules), the ones that reference the first matched symbols outside
// their own file. Capped and read within repo_map's byte budget.
func (e *Engine) relatedFiles(ctx context.Context, r *os.Root, candidates []string, symbols []mapSymbol) []relatedRef {
	targets := map[string]string{} // name -> defining file
	var order []string
	for _, s := range symbols {
		if len(order) == relatedSymbols {
			break
		}
		if _, dup := targets[s.Name]; dup || len(s.Name) < 3 {
			continue
		}
		targets[s.Name] = s.Path
		order = append(order, s.Name)
	}
	out := []relatedRef{}
	seen := map[string]bool{}
	read := 0
	for _, path := range candidates {
		if ctx.Err() != nil || len(out) >= relatedFiles || read >= mapMaxBytes {
			break
		}
		data, err := mapRead(r, path)
		if err != nil {
			continue
		}
		read += len(data)
		ids := e.identifiers(path, data)
		for _, name := range order {
			if targets[name] == path || seen[path] {
				continue
			}
			if line, ok := ids.lines[name]; ok {
				seen[path] = true
				out = append(out, relatedRef{Path: path, Line: line, Symbol: name, DefinedIn: targets[name], Provenance: "references " + name + " (lexical match, not type-checked)"})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
