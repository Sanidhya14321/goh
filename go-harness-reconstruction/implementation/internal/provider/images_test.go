package provider

import (
	"encoding/json"
	"strings"
	"testing"

	"example.com/independent-harness/internal/core"
)

func imageHistory() []core.Message {
	return []core.Message{{Role: "system", Content: "s"},
		{Role: "user", Content: "see this", Images: []core.Image{{Path: "a.png", MediaType: "image/png", Data: "iVBORw0K"}}}}
}

func TestEncodersSendImages(t *testing.T) {
	cfg := core.Defaults()
	cfg.Model = "fixture"
	wire := chatWireMessages(imageHistory())
	b := mustJSON(t, wire)
	if !strings.Contains(b, `"type":"image_url"`) || !strings.Contains(b, "data:image/png;base64,iVBORw0K") || strings.Contains(b, `"images"`) {
		t.Fatal("chat", b)
	}
	r, err := encodeResponses(cfg, imageHistory(), nil)
	if err != nil || !strings.Contains(string(r), `"type":"input_image"`) || !strings.Contains(string(r), "data:image/png;base64,iVBORw0K") {
		t.Fatal("responses", err, string(r))
	}
	a, err := EncodeAnthropic(cfg, imageHistory(), nil)
	if err != nil || !strings.Contains(string(a), `"type":"image"`) || !strings.Contains(string(a), `"media_type":"image/png"`) {
		t.Fatal("anthropic", err, string(a))
	}
	plain := chatWireMessages([]core.Message{{Role: "user", Content: "x"}})
	if _, ok := plain[0].(core.Message); !ok {
		t.Fatal("messages without images must be sent unchanged")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
