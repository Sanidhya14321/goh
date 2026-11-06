package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestRuntimeToolSchemasCompile(t *testing.T) {
	a, _ := testAgent(t)
	for _, d := range a.localDefinitions() {
		c := jsonschema.NewCompiler()
		uri := fmt.Sprintf("https://harness.invalid/%s", d.Function.Name)
		wire, e := json.Marshal(d.Function.Parameters)
		if e != nil {
			t.Fatal(e)
		}
		var document any
		if e = json.Unmarshal(wire, &document); e != nil {
			t.Fatal(e)
		}
		if e := c.AddResource(uri, document); e != nil {
			t.Fatal(e)
		}
		if _, e := c.Compile(uri); e != nil {
			t.Fatalf("%s: %v", d.Function.Name, e)
		}
	}
	if status, e := a.Compact(context.Background()); status != "failed" || e == nil {
		t.Fatal("empty compact changed")
	}
}
