package structured

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestStrictSchemaAndLosslessJSONOutput(t *testing.T) {
	schema, err := Compile(json.RawMessage(`{"type":"object","properties":{"count":{"type":"integer","minimum":9007199254740993},"ok":{"type":"boolean"}},"required":["count","ok"],"additionalProperties":false}`))
	if err != nil {
		t.Fatal(err)
	}
	value, err := Validate(schema, `{"count":9007199254740993,"ok":true}`)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(value)
	if !strings.Contains(string(encoded), "9007199254740993") {
		t.Fatal(string(encoded))
	}
	for _, text := range []string{`{"count":9007199254740992,"ok":true}`, `{"count":1,"ok":"yes"}`, `{"count":9007199254740993,"ok":true,"extra":1}`, "```json\n{}\n```", `{} {}`, `{"count":1,"count":9007199254740993,"ok":true}`} {
		if _, err = Validate(schema, text); err == nil {
			t.Fatal("accepted", text)
		}
	}
}
func TestSchemaPreflightNeverFetchesAndBounds(t *testing.T) {
	calls := atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	for _, raw := range []string{`false`, `null`, `[]`, `{"type":"unknown"}`, `{"$ref":"` + server.URL + `"}`, `{"$dynamicRef":"#"}`, `{"$schema":"` + server.URL + `"}`, `{"type":"object","type":"string"}`, strings.Repeat(" ", MaxSchema+1) + `{}`, strings.Repeat(`{"properties":{"x":`, 20) + `{}` + strings.Repeat(`}}`, 20)} {
		if _, err := Compile(json.RawMessage(raw)); err == nil {
			t.Fatal("accepted schema", raw)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("schema fetched remote URL")
	}
	schema, err := Compile(json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`"` + strings.Repeat("<", MaxOutput/2) + `"`, `"` + strings.Repeat("a", MaxOutput) + `"`, strings.Repeat(`[`, 40) + `0` + strings.Repeat(`]`, 40)} {
		if _, err = Validate(schema, raw); err == nil {
			t.Fatal("accepted unbounded result")
		}
	}
}
func FuzzStructuredResult(f *testing.F) {
	f.Add(`{"ok":true}`)
	f.Add(`{"a":1,"a":2}`)
	schema, err := Compile(json.RawMessage(`{}`))
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		v, err := Validate(schema, raw)
		if err != nil {
			return
		}
		encoded, err := json.Marshal(v)
		if err != nil || len(encoded) > MaxOutput {
			t.Fatal("invalid accepted output")
		}
	})
}
