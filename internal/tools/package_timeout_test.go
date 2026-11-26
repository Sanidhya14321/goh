package tools

import (
	"encoding/json"
	"testing"
)

func TestPackageManagerTimeout(t *testing.T) {
	for _, c := range []struct {
		command string
		limit   int
		want    int
	}{
		{"apt-get update -qq && apt-get install -y -qq tesseract-ocr", 20, minPackageTimeout},
		{"sudo DEBIAN_FRONTEND=noninteractive /usr/bin/apt install -y jq", 0, minPackageTimeout},
		{"dpkg --configure -a", 30, minPackageTimeout},
		{"apk add curl", 450, 450},
		{"pip install pillow", 20, 20},
		{"echo apt-getter", 20, 20},
		{"go test ./...", 0, 120},
	} {
		raw, _ := json.Marshal(map[string]any{"command": c.command})
		if c.limit > 0 {
			raw, _ = json.Marshal(map[string]any{"command": c.command, "timeout_seconds": c.limit})
		}
		a, err := decode("shell", raw)
		if err != nil {
			t.Fatal(err)
		}
		if a.Timeout != c.want {
			t.Errorf("%q: timeout %d, want %d", c.command, a.Timeout, c.want)
		}
	}
}
