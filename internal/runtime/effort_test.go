package runtime

import (
	"testing"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
)

func TestSetReasoningEffort(t *testing.T) {
	a := &Agent{Config: core.Config{Protocol: "openai_responses", Model: "m"}}
	a.Provider = provider.New(a.Config)
	if err := a.SetReasoningEffort("high"); err != nil || a.Config.ReasoningEffort != "high" {
		t.Fatal(err, a.Config.ReasoningEffort)
	}
	if c, ok := a.Provider.(*provider.Client); !ok || c.Config.ReasoningEffort != "high" {
		t.Fatal("provider not rebuilt with the new effort")
	}
	if err := a.SetReasoningEffort("extreme"); err == nil || a.Config.ReasoningEffort != "high" {
		t.Fatal("invalid effort accepted or state changed")
	}
	if err := a.SetReasoningEffort(""); err != nil || a.Config.ReasoningEffort != "" {
		t.Fatal(err)
	}
}
