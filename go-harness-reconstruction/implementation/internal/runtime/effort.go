package runtime

import "example.com/independent-harness/internal/provider"

// SetReasoningEffort changes reasoning effort for this agent's later turns
// ("" turns it off). Call it only between runs. The provider client is
// rebuilt only when it is the stock client, so injected models stay.
func (a *Agent) SetReasoningEffort(effort string) error {
	next := a.Config
	next.ReasoningEffort = effort
	if err := next.ValidateReasoningEffort(); err != nil {
		return err
	}
	a.Config = next
	if _, stock := a.Provider.(*provider.Client); stock {
		a.Provider = provider.New(next)
	}
	return nil
}
