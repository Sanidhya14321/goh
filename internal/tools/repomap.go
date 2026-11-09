package tools

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Sanidhya14321/goh/internal/platform"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const mapMaxFile = 256 << 10
const mapMaxBytes = 8 << 20
const mapMaxFiles = 256
const mapMaxSymbols = 2000
const mapCacheEntries = 32
const mapMaxResult = 32 << 10

type mapArgs struct {
	Path  string `json:"path,omitempty"`
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}
type mapSymbol struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Signature string `json:"signature"`
	SHA       string `json:"sha256"`
}
type mapCacheEntry struct {
	Digest     string
	Symbols    []mapSymbol
	ParseError bool
	Partial    bool
}

func decodeMap(raw json.RawMessage) (mapArgs, error) {
	var a mapArgs
	if len(raw) > 8192 {
		return a, errors.New("repo_map arguments exceed 8KiB")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return a, errors.New("repo_map arguments must be an object")
	}
	for _, v := range obj {
		if strings.TrimSpace(string(v)) == "null" {
			return a, errors.New("repo_map arguments cannot be null")
		}
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return a, err
	}
	p, err := clean(a.Path, true)
	if err != nil {
		return a, err
	}
	a.Path = p
	if len(a.Path) > 4096 || len(a.Query) > 256 || !utf8.ValidString(a.Query) || a.Limit < 0 || a.Limit > 500 {
		return a, errors.New("repo_map needs query<=256 bytes and limit1..500")
	}
	if _, ok := obj["limit"]; ok && a.Limit == 0 {
		return a, errors.New("repo_map limit must be positive")
	}
	if a.Limit == 0 {
		a.Limit = 200
	}
	return a, nil
}
func (e *Engine) prepareRepoMap(raw json.RawMessage) (string, bool, error) {
	a, err := decodeMap(raw)
	if err != nil {
		return "", false, err
	}
	r, err := os.OpenRoot(e.root)
	if err != nil {
		return "", false, err
	}
	defer r.Close()
	if err = checked(r, a.Path, false); err != nil {
		return "", false, err
	}
	st, err := r.Stat(a.Path)
	if err != nil {
		return "", false, err
	}
	if !st.IsDir() && !mapSupported(a.Path) {
		return "", false, errors.New("repo_map supports Go, Python, JavaScript/TypeScript, Rust, Java/Kotlin/C#/Scala, C/C++ and Ruby files, or project directories")
	}
	return "Inspect bounded declarations under " + a.Path, false, nil
}
func mapRead(r *os.Root, path string) ([]byte, error) {
	if err := checked(r, path, false); err != nil {
		return nil, err
	}
	f, err := r.OpenFile(path, os.O_RDONLY|platform.NonBlock|platform.NoFollow, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > mapMaxFile {
		return nil, errors.New("repo map skips nonregular files and files over256KiB")
	}
	b, err := io.ReadAll(io.LimitReader(f, mapMaxFile+1))
	if err != nil {
		return nil, err
	}
	if len(b) > mapMaxFile || !utf8.Valid(b) {
		return nil, errors.New("unsupported Go source size/encoding")
	}
	return b, nil
}
func shortSignature(s string) string {
	if len(s) > 512 {
		s = s[:512]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
		s += "…"
	}
	return s
}
func parseSymbols(path string, data []byte) mapCacheEntry {
	if !strings.HasSuffix(path, ".go") {
		return regexSymbols(path, data)
	}
	digest := hash(data)
	out := mapCacheEntry{Digest: digest}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, parser.SkipObjectResolution)
	if err != nil {
		out.ParseError = true
		return out
	}
	add := func(pos token.Pos, kind, name, sig string) {
		if len(out.Symbols) >= mapMaxSymbols {
			out.Partial = true
			return
		}
		out.Symbols = append(out.Symbols, mapSymbol{Path: path, Line: fset.Position(pos).Line, Kind: kind, Name: name, Signature: shortSignature(sig), SHA: digest})
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			signature := *d
			signature.Body = nil
			signature.Doc = nil
			var b strings.Builder
			if err := printer.Fprint(&b, fset, &signature); err != nil {
				continue
			}
			kind, name := "function", d.Name.Name
			if d.Recv != nil {
				kind = "method"
				var recv strings.Builder
				printer.Fprint(&recv, fset, d.Recv.List[0].Type)
				name = recv.String() + "." + name
			}
			add(d.Pos(), kind, name, b.String())
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch x := spec.(type) {
				case *ast.TypeSpec:
					kind := "type"
					switch x.Type.(type) {
					case *ast.StructType:
						kind = "struct"
					case *ast.InterfaceType:
						kind = "interface"
					}
					add(x.Pos(), kind, x.Name.Name, "type "+x.Name.Name+" ("+kind+")")
				case *ast.ValueSpec:
					for _, name := range x.Names {
						if name.Name != "_" {
							add(name.Pos(), d.Tok.String(), name.Name, d.Tok.String()+" "+name.Name)
						}
					}
				}
			}
		}
		if out.Partial {
			break
		}
	}
	return out
}
func (e *Engine) symbols(path string, data []byte) (mapCacheEntry, bool) {
	digest := hash(data)
	e.mapMu.Lock()
	defer e.mapMu.Unlock()
	if cached, ok := e.mapCache[path]; ok && cached.Digest == digest {
		return cached, true
	}
	// Hash identity, not mtime, invalidates same-size/same-time external edits.
	parsed := parseSymbols(path, data)
	if e.mapCache == nil {
		e.mapCache = map[string]mapCacheEntry{}
	}
	if _, ok := e.mapCache[path]; !ok {
		if len(e.mapOrder) >= mapCacheEntries {
			delete(e.mapCache, e.mapOrder[0])
			e.mapOrder = e.mapOrder[1:]
		}
		e.mapOrder = append(e.mapOrder, path)
	}
	e.mapCache[path] = parsed
	return parsed, false
}
func (e *Engine) repoMap(ctx context.Context, raw json.RawMessage) Result {
	if _, _, err := e.prepareRepoMap(raw); err != nil {
		return failure(err)
	}
	a, _ := decodeMap(raw)
	r, err := os.OpenRoot(e.root)
	if err != nil {
		return failure(err)
	}
	defer r.Close()
	symbols := []mapSymbol{}
	visited, files, bytesRead, skipped, parseErrors, hits := 0, 0, 0, 0, 0, 0
	truncated := false
	query := strings.ToLower(a.Query)
	err = fs.WalkDir(r.FS(), filepath.ToSlash(a.Path), func(path string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		visited++
		if visited > 20000 {
			truncated = true
			return fs.SkipAll
		}
		if walkErr != nil {
			skipped++
			return nil
		}
		if e.denied(path) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", ".harness", "vendor":
				if path != a.Path {
					return fs.SkipDir
				}
			}
			return nil
		}
		if !mapSupported(path) || !d.Type().IsRegular() {
			return nil
		}
		if files >= mapMaxFiles || bytesRead >= mapMaxBytes || len(symbols) >= mapMaxSymbols {
			truncated = true
			return fs.SkipAll
		}
		data, err := mapRead(r, path)
		if err != nil {
			skipped++
			return nil
		}
		if bytesRead+len(data) > mapMaxBytes {
			truncated = true
			return fs.SkipAll
		}
		files++
		bytesRead += len(data)
		parsed, cached := e.symbols(path, data)
		if cached {
			hits++
		}
		if parsed.ParseError {
			parseErrors++
			return nil
		}
		if parsed.Partial {
			truncated = true
		}
		for _, symbol := range parsed.Symbols {
			if query != "" && !strings.Contains(strings.ToLower(symbol.Name+" "+path), query) {
				continue
			}
			if len(symbols) >= mapMaxSymbols {
				truncated = true
				break
			}
			symbols = append(symbols, symbol)
		}
		return nil
	})
	if err != nil {
		return failure(err)
	}
	sort.SliceStable(symbols, func(i, j int) bool {
		if query != "" {
			a, b := strings.EqualFold(symbols[i].Name, query), strings.EqualFold(symbols[j].Name, query)
			if a != b {
				return a
			}
		}
		if symbols[i].Path != symbols[j].Path {
			return symbols[i].Path < symbols[j].Path
		}
		return symbols[i].Line < symbols[j].Line
	})
	matched := len(symbols)
	if len(symbols) > a.Limit {
		symbols = symbols[:a.Limit]
		truncated = true
	}
	output := map[string]any{"language": "go", "symbols": symbols, "matched_symbols": matched, "files_scanned": files, "bytes_read": bytesRead, "parse_cache_hits": hits, "skipped_files": skipped, "parse_errors": parseErrors, "truncated": truncated, "scope": "Syntax declarations only; no type checking, graph ranking, compilation or instruction authority."}
	for {
		output["symbols"] = symbols
		output["truncated"] = truncated
		b, err := json.Marshal(output)
		if err != nil {
			return failure(err)
		}
		if len(b) <= mapMaxResult {
			return Result{Content: string(b)}
		}
		if len(symbols) == 0 {
			return failure(errors.New("repo map metadata exceeds budget"))
		}
		symbols = symbols[:len(symbols)-1]
		truncated = true
	}
}
