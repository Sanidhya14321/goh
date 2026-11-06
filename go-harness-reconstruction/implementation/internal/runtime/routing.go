package runtime

import (
	"errors"
	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/provider"
)

func (a *Agent) modelForRole(role string) (Model, core.Config, error) {
	if _, ok := a.Config.ModelRoles[role]; !ok {
		return a.Provider, a.Config, nil
	}
	config, err := a.Config.WithModelRole(role)
	if err != nil {
		return nil, config, err
	}
	if err = config.Validate(); err != nil {
		return nil, config, err
	}
	model := Model(provider.New(config))
	if a.RoleModelFactory != nil {
		model = a.RoleModelFactory(config, role)
		if model == nil {
			return nil, config, errors.New("model role factory returned no provider")
		}
	}
	data := map[string]any{"purpose": role, "model": config.Model, "profile": config.Profile, "protocol": config.EffectiveProtocol(), "context_window": config.ContextWindow}
	if err = a.Store.Append("model_route", data); err != nil {
		return nil, config, err
	}
	a.event("model_route", data)
	return model, config, nil
}
