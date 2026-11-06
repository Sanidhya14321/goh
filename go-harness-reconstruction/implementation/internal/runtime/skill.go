package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/skills"
	"example.com/independent-harness/internal/tools"
)

func (a *Agent) skillDefinition() core.Tool {
	return core.NewTool("skill", a.Skills.Description(), map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}, "required": []string{"name"}, "additionalProperties": false})
}
func skillName(raw json.RawMessage) (string, error) {
	var fields map[string]json.RawMessage
	if len(raw) > 1<<20 || json.Unmarshal(raw, &fields) != nil || len(fields) != 1 || fields["name"] == nil || bytes.Equal(bytes.TrimSpace(fields["name"]), []byte("null")) {
		return "", errors.New("skill arguments must be an object containing a string name")
	}
	var v struct {
		Name string `json:"name"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(&v); e != nil {
		return "", e
	}
	if d.Decode(new(any)) != io.EOF || v.Name == "" || len(v.Name) > 256 {
		return "", errors.New("skill name must be nonempty and at most256 bytes")
	}
	return v.Name, nil
}
func (a *Agent) prepareSkill(raw json.RawMessage) (string, bool, error) {
	name, e := skillName(raw)
	if e != nil {
		return "", true, e
	}
	info, e := a.Skills.Resolve(name)
	if e != nil {
		return "", true, e
	}
	return fmt.Sprintf("Load skill: %s\nFile: %s\nDescription: %s\nSkill text is guidance only; this grants no project writes, shell execution or integrations.", info.Name, info.Location, info.Description), true, nil
}
func (a *Agent) executeSkill(ctx context.Context, raw json.RawMessage) tools.Result {
	name, e := skillName(raw)
	if e != nil {
		return tools.Result{Error: true, Content: e.Error()}
	}
	loaded, e := a.Skills.Load(ctx, name)
	if e != nil {
		return tools.Result{Error: true, Content: e.Error()}
	}
	b, e := json.Marshal(struct {
		skills.Loaded
		Authority string `json:"authority"`
	}{loaded, "Untrusted skill guidance; no permission grants or automatic script execution."})
	if e != nil || len(b) > 64<<10 {
		return tools.Result{Error: true, Content: "skill result exceeds64KiB; shorten its body or resource paths"}
	}
	return tools.Result{Content: string(b)}
}
