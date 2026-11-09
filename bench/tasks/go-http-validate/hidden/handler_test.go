package task

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func call(method, body string) (int, map[string]any) {
	req := httptest.NewRequest(method, "/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	UsersHandler(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestUsersHandler(t *testing.T) {
	if code, out := call("POST", `{"name":"Ada","age":36}`); code != 201 || out["name"] != "Ada" {
		t.Fatal(code, out)
	}
	for _, bad := range []string{`{`, `{"name":"","age":3}`, `{"name":"x","age":-1}`, `{"name":"x","age":151}`} {
		if code, out := call("POST", bad); code != 400 || out["error"] == nil {
			t.Fatalf("%s: %d %v", bad, code, out)
		}
	}
	if code, _ := call("GET", ""); code != 405 {
		t.Fatal(code)
	}
}
