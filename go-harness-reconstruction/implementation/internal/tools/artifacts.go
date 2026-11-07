package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"example.com/independent-harness/internal/platform"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

var artifactName = regexp.MustCompile(`^shell-[0-9]+\.log$`)

type artifactArgs struct {
	ID       string `json:"id"`
	Offset   int64  `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Tail     bool   `json:"tail,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

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
