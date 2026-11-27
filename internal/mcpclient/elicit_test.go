package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const deployForm = `{"type":"object","properties":{
 "env":{"type":"string","title":"Environment","enum":["staging","prod"],"enumNames":["Staging","Production"]},
 "replicas":{"type":"integer","description":"How many"},
 "confirm":{"type":"boolean","title":"Confirm"},
 "regions":{"type":"array","items":{"anyOf":[{"const":"eu","title":"Europe"},{"const":"us","title":"US"}]}},
 "note":{"type":"string"}},
 "required":["env","replicas","confirm"]}`

func elicitServer(t *testing.T) *httptest.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "deployer", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "deploy", Description: "Deploy"}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		var schema any
		json.Unmarshal([]byte(deployForm), &schema)
		res, err := req.Session.Elicit(ctx, &mcp.ElicitParams{Message: "Deploy settings", RequestedSchema: schema})
		if err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "elicit error: " + err.Error()}}}, nil, nil
		}
		b, _ := json.Marshal(res)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
	})
	ts := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	t.Cleanup(ts.Close)
	return ts
}

func callDeploy(t *testing.T, ask AskFunc) string {
	t.Helper()
	ts := elicitServer(t)
	m := New(map[string]core.MCPConfig{"ops": {URL: ts.URL, HeadersEnv: map[string]string{"Authorization": "ELICIT_TOKEN"}}}, t.TempDir(), allowAll)
	t.Setenv("ELICIT_TOKEN", "Bearer x")
	ForAgent(m, ask != nil, ask)
	defer m.Close()
	if err := m.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	r := m.Execute(context.Background(), "mcp__ops__deploy", json.RawMessage(`{}`))
	if r.Error {
		t.Fatal(r.Content)
	}
	return r.Content
}

func TestElicitationAsksTheUser(t *testing.T) {
	var asked core.QuestionRequest
	out := callDeploy(t, func(_ context.Context, r core.QuestionRequest) (core.QuestionReply, error) {
		asked = r
		return core.QuestionReply{Answers: [][]string{{"Yes"}, {"Production"}, {"3"}, {}, {"Europe", "US"}}}, nil
	})
	if len(asked.Questions) != 5 || asked.Questions[0].Header != "MCP ops" || !strings.Contains(asked.Questions[0].Question, "Deploy settings") ||
		!strings.Contains(asked.Questions[0].Question, "sent to it") || !asked.Questions[4].Multiple || len(asked.Questions[2].Options) != 0 ||
		!strings.Contains(asked.Questions[3].Question, "optional") || strings.Contains(asked.Questions[1].Question, "optional") {
		t.Fatalf("questions %+v", asked)
	}
	want := `\"action\":\"accept\",\"content\":{\"confirm\":true,\"env\":\"prod\",\"regions\":[\"eu\",\"us\"],\"replicas\":3}`
	if !strings.Contains(out, want) {
		t.Fatalf("result %s", out)
	}
}

func TestElicitationDeclined(t *testing.T) {
	out := callDeploy(t, func(context.Context, core.QuestionRequest) (core.QuestionReply, error) {
		return core.QuestionReply{}, errors.New("dismissed")
	})
	if !strings.Contains(out, `\"action\":\"decline\"`) {
		t.Fatal(out)
	}
	// A wrong type is declined rather than sent.
	out = callDeploy(t, func(context.Context, core.QuestionRequest) (core.QuestionReply, error) {
		return core.QuestionReply{Answers: [][]string{{"Yes"}, {"Production"}, {"three"}, {}, {}}}, nil
	})
	if !strings.Contains(out, `\"action\":\"decline\"`) {
		t.Fatal(out)
	}
	// Nobody to ask: elicitation is not offered at all.
	if out := callDeploy(t, nil); !strings.Contains(out, "elicit error") {
		t.Fatal(out)
	}
}

func TestElicitFields(t *testing.T) {
	var schema any
	json.Unmarshal([]byte(deployForm), &schema)
	fields, err := elicitFields(schema)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, f := range fields {
		keys = append(keys, f.key)
	}
	if !reflect.DeepEqual(keys, []string{"confirm", "env", "replicas", "note", "regions"}) {
		t.Fatal("order", keys)
	}
	for _, bad := range []string{`{"type":"array"}`, `{"type":"object","properties":{}}`, `{"type":"object","properties":{"a":{"type":"object"}}}`,
		`{"type":"object","properties":{"a":{"type":"string","enum":["x","x"]}}}`} {
		var s any
		json.Unmarshal([]byte(bad), &s)
		if _, err := elicitFields(s); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	if _, err := elicitContent(fields, core.QuestionReply{Answers: [][]string{{"No"}, {}, {"1"}, {}, {}}}); err == nil {
		t.Fatal("missing required answer accepted")
	}
}
