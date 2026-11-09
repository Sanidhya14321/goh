package runtime

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
)

const (
	maxImages          = 4
	maxImageBytes      = 5 << 20
	maxTotalImageBytes = 8 << 20
	imageTokenEstimate = 1600
)

var imageTypes = map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp"}

// promptImages attaches images named as @path in the prompt, as OpenCode does
// for pasted or mentioned screenshots. Paths must be inside the project and
// are read through os.Root, so symlinks and escapes are refused. The file's
// leading bytes must match its extension.
func (a *Agent) promptImages(prompt string) ([]core.Image, error) {
	var paths []string
	seen := map[string]bool{}
	for _, field := range strings.Fields(prompt) {
		path, ok := strings.CutPrefix(field, "@")
		path = strings.TrimRight(path, ".,;:!?)\"'")
		if !ok || imageTypes[strings.ToLower(filepath.Ext(path))] == "" || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	if len(paths) > maxImages {
		return nil, fmt.Errorf("at most %d images can be attached to one prompt", maxImages)
	}
	root, err := os.OpenRoot(a.Store.Project)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	var images []core.Image
	total := 0
	for _, path := range paths {
		clean := filepath.Clean(filepath.FromSlash(path))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("@%s: images must be inside the project", path)
		}
		info, err := root.Lstat(clean)
		if err != nil {
			return nil, fmt.Errorf("@%s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("@%s: not a regular file", path)
		}
		if info.Size() > maxImageBytes {
			return nil, fmt.Errorf("@%s: images are limited to 5 MiB", path)
		}
		f, err := root.Open(clean)
		if err != nil {
			return nil, fmt.Errorf("@%s: %w", path, err)
		}
		b, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("@%s: %w", path, err)
		}
		if len(b) > maxImageBytes {
			return nil, fmt.Errorf("@%s: images are limited to 5 MiB", path)
		}
		mediaType := imageTypes[strings.ToLower(filepath.Ext(clean))]
		if !imageMagicMatches(mediaType, b) {
			return nil, fmt.Errorf("@%s: content is not a %s image", path, mediaType)
		}
		if total += len(b); total > maxTotalImageBytes {
			return nil, fmt.Errorf("attached images exceed 8 MiB in total")
		}
		images = append(images, core.Image{Path: filepath.ToSlash(clean), MediaType: mediaType, Data: base64.StdEncoding.EncodeToString(b)})
	}
	return images, nil
}

func imageMagicMatches(mediaType string, b []byte) bool {
	switch mediaType {
	case "image/png":
		return strings.HasPrefix(string(b), "\x89PNG\r\n\x1a\n")
	case "image/jpeg":
		return len(b) > 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF
	case "image/gif":
		return strings.HasPrefix(string(b), "GIF87a") || strings.HasPrefix(string(b), "GIF89a")
	case "image/webp":
		return len(b) > 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP"
	}
	return false
}

// withoutImages returns messages with image data replaced by a path note, and
// the number of images removed. Used for size estimates and summaries.
func withoutImages(ms []core.Message) ([]core.Message, int) {
	count := 0
	var out []core.Message
	for i, m := range ms {
		if len(m.Images) == 0 {
			continue
		}
		if out == nil {
			out = append([]core.Message(nil), ms...)
		}
		names := make([]string, len(m.Images))
		for j, img := range m.Images {
			names[j] = img.Path
		}
		count += len(m.Images)
		out[i].Images = nil
		out[i].Content += "\n[attached images: " + strings.Join(names, ", ") + "]"
	}
	if out == nil {
		return ms, 0
	}
	return out, count
}
