package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var artifactName = regexp.MustCompile(`^(shell|fetch)-[0-9]+\.log$`)

type artifactArgs struct {
	ID       string `json:"id"`
	Offset   int64  `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Tail     bool   `json:"tail,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	// Pattern (RE2) returns the matching lines with their byte offsets
	// instead of a byte range: a long log or fetched page is searched
	// without reading it all into context.
	Pattern string `json:"pattern,omitempty"`
}

const maxPatternLine = 300

func decodeArtifact(raw json.RawMessage) (artifactArgs, error) {
	var a artifactArgs
	if len(raw) > 4096 {
		return a, errors.New("artifact arguments exceed 4KiB")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return a, errors.New("artifact arguments must be an object")
	}
	for _, v := range object {
		if string(v) == "null" {
			return a, errors.New("artifact arguments cannot be null")
		}
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(&a); err != nil {
		return a, err
	}
	if !artifactName.MatchString(a.ID) || len(a.ID) > 64 || a.Offset < 0 || a.Limit < 0 || a.Limit > 8192 {
		return a, errors.New("invalid artifact ID, offset or limit (1..8192 bytes)")
	}
	if a.Tail && a.Offset != 0 {
		return a, errors.New("tail and offset cannot be combined")
	}
	if _, ok := object["pattern"]; ok {
		if a.Pattern == "" || len(a.Pattern) > 256 || a.Tail || a.Encoding == "base64" {
			return a, errors.New("pattern must be a nonempty RE2 expression of at most 256 bytes, without tail or base64")
		}
		if _, err := regexp.Compile(a.Pattern); err != nil {
			return a, errors.New("invalid pattern: " + err.Error())
		}
	}
	if _, ok := object["limit"]; ok && a.Limit == 0 {
		return a, errors.New("artifact limit must be positive")
	}
	if a.Limit == 0 {
		a.Limit = 4096
	}
	if a.Encoding == "" {
		a.Encoding = "text"
	}
	if a.Encoding != "text" && a.Encoding != "base64" {
		return a, errors.New("artifact encoding must be text or base64")
	}
	return a, nil
}

// SaveArtifact retains text another tool produced (a fetched page) as a
// session-local artifact that read_artifact can page through, and returns
// its ID. kind names the producer ("fetch").
func (e *Engine) SaveArtifact(kind, text string) (string, error) {
	if kind != "fetch" {
		return "", errors.New("unknown artifact kind")
	}
	if len(text) > maxArtifact {
		text = strings.ToValidUTF8(text[:maxArtifact], "")
	}
	if err := os.MkdirAll(e.artifacts, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(e.artifacts, 0o700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(e.artifacts, kind+"-*.log")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err = f.Chmod(0o600); err != nil {
		return "", err
	}
	if _, err = io.WriteString(f, text); err != nil {
		return "", err
	}
	return filepath.Base(f.Name()), f.Close()
}
func (e *Engine) prepareArtifact(raw json.RawMessage) (string, bool, error) {
	a, err := decodeArtifact(raw)
	if err != nil {
		return "", false, err
	}
	return "Read retained output artifact " + a.ID + " from this session only", false, nil
}
func (e *Engine) readArtifact(ctx context.Context, raw json.RawMessage) Result {
	a, err := decodeArtifact(raw)
	if err != nil {
		return failure(err)
	}
	if err = ctx.Err(); err != nil {
		return failure(err)
	}
	root, err := os.OpenRoot(e.artifacts)
	if err != nil {
		return failure(err)
	}
	defer root.Close()
	if err = checked(root, a.ID, false); err != nil {
		return failure(err)
	}
	f, err := root.OpenFile(a.ID, os.O_RDONLY|platform.NonBlock|platform.NoFollow, 0)
	if err != nil {
		return failure(err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return failure(err)
	}
	if !st.Mode().IsRegular() || st.Size() > maxArtifact {
		return failure(errors.New("artifact must be a retained regular output file of at most 1MiB"))
	}
	// Hash exactly the retained artifact, never an arbitrary model-supplied path.
	data, err := io.ReadAll(io.LimitReader(f, maxArtifact+1))
	if err != nil {
		return failure(err)
	}
	if len(data) > maxArtifact {
		return failure(errors.New("artifact grew past retained limit"))
	}
	if err = ctx.Err(); err != nil {
		return failure(err)
	}
	if a.Pattern != "" {
		return result(artifactMatches(a, data))
	}
	offset := a.Offset
	if a.Tail {
		offset = int64(len(data)) - int64(a.Limit)
		if offset < 0 {
			offset = 0
		}
	}
	if offset > int64(len(data)) {
		return failure(errors.New("artifact offset exceeds retained length"))
	}
	end := min(int64(len(data)), offset+int64(a.Limit))
	part := data[offset:end]
	content := string(part)
	repaired := false
	if a.Encoding == "base64" {
		content = base64.StdEncoding.EncodeToString(part)
	} else if !utf8.Valid(part) {
		repaired = true
		content = strings.ToValidUTF8(content, "�")
	}
	return result(map[string]any{"id": a.ID, "content": content, "encoding": a.Encoding, "utf8_repaired": repaired, "offset": offset, "next_offset": end, "bytes": len(part), "retained_bytes": len(data), "sha256": hash(data), "eof": end == int64(len(data)), "scope": "Retained session output only; discarded bytes cannot be recovered."})
}

// artifactMatches lists the lines of data from a.Offset on that match a.Pattern,
// each with its byte offset and cut to maxPatternLine bytes, until a.Limit
// bytes of matches are listed.
func artifactMatches(a artifactArgs, data []byte) map[string]any {
	re := regexp.MustCompile(a.Pattern) // validated in decodeArtifact
	start := min(a.Offset, int64(len(data)))
	var b strings.Builder
	matches, listed := 0, 0
	next := int64(len(data))
	for pos := start; pos < int64(len(data)); {
		end := int64(len(data))
		if i := bytes.IndexByte(data[pos:], '\n'); i >= 0 {
			end = pos + int64(i)
		}
		line := data[pos:end]
		if re.Match(line) {
			matches++
			text := line
			if len(text) > maxPatternLine {
				text = text[:maxPatternLine]
			}
			entry := fmt.Sprintf("%d: %s\n", pos, strings.ToValidUTF8(string(text), "�"))
			if b.Len()+len(entry) > a.Limit {
				if next == int64(len(data)) {
					next = pos
				}
			} else {
				b.WriteString(entry)
				listed++
			}
		}
		pos = end + 1
	}
	return map[string]any{"id": a.ID, "pattern": a.Pattern, "matches": matches, "listed": listed, "content": b.String(), "next_offset": next, "retained_bytes": len(data), "sha256": hash(data),
		"scope": "Matching lines as \"byte_offset: line\" (cut to 300 bytes); read a range with offset and limit. Retained session output only."}
}
