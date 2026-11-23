package runtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
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
	for _, path := range mentions(prompt) {
		if imageTypes[strings.ToLower(filepath.Ext(path))] == "" || seen[path] {
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
		if a.pathDenied(clean) {
			return nil, fmt.Errorf("@%s: access to this path is denied by the user's file_rules", path)
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

// requestImageMargin is room left in the encoded request limit for tool
// definitions and protocol framing, which the projection does not see.
const requestImageMargin = 1 << 20

// boundImageHistory keeps the encoded request under the provider's byte
// limit. Token estimates count a fixed allowance per image, which is right
// for the token dimension, but replaying every earlier attachment's base64
// data can exceed the transport limit while the token estimate is small.
// Images on the latest user message are always kept; older image data is
// dropped newest-last to a path note until the rest fits. The journal and
// a.History keep every image. Returns the number of images dropped.
func boundImageHistory(history []core.Message) ([]core.Message, int) {
	total := 0
	for _, m := range history {
		for _, img := range m.Images {
			total += len(img.Data)
		}
	}
	if total <= provider.MaxRequestBytes/2 {
		return history, 0 // the text alone would need 8 MiB to matter
	}
	plain, _ := withoutImages(history)
	text, _ := json.Marshal(plain)
	budget := provider.MaxRequestBytes - requestImageMargin - len(text)
	current := -1
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" && !history[i].Automatic {
			current = i
			break
		}
	}
	used := 0
	if current >= 0 {
		for _, img := range history[current].Images {
			used += len(img.Data)
		}
	}
	var out []core.Message
	dropped := 0
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if i == current || len(m.Images) == 0 {
			continue
		}
		size := 0
		for _, img := range m.Images {
			size += len(img.Data)
		}
		if dropped == 0 && used+size <= budget {
			used += size
			continue
		}
		if out == nil {
			out = append([]core.Message(nil), history...)
		}
		names := make([]string, len(m.Images))
		for j, img := range m.Images {
			names[j] = img.Path
		}
		dropped += len(m.Images)
		out[i].Images = nil
		out[i].Content += "\n[earlier attached images not resent, to fit the provider request size: " + strings.Join(names, ", ") + "]"
	}
	if out == nil {
		return history, 0
	}
	return out, dropped
}
