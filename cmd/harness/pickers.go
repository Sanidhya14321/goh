package main

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/Sanidhya14321/goh/internal/provider"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tui"
)

func modelOptions(cfg core.Config) []tui.Option {
	options := []tui.Option{{Value: "", Label: "Default profile", Description: "Use configured provider defaults"}}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		profile := cfg.Profiles[name]
		description := "Inherit model settings"
		if profile.Model != nil {
			description = *profile.Model
		}
		options = append(options, tui.Option{Value: name, Label: name, Description: description})
	}
	return append(options, tui.Option{Value: "@provider", Label: "Connect a provider", Description: "Configure API URL, model and key"})
}

func agentOptions(cfg core.Config) []tui.Option {
	options := []tui.Option{
		{Value: "build", Label: "Build", Description: "Investigate and edit with approvals"},
		{Value: "plan", Label: "Plan", Description: "Read-only project investigation"},
	}
	names := make([]string, 0, len(cfg.Agents))
	for name := range cfg.Agents {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		profile := cfg.Agents[name]
		options = append(options, tui.Option{Value: name, Label: name, Description: profile.Mode + " · " + profile.Description})
	}
	return options
}

func nextAgent(cfg core.Config, previous bool) string {
	options := agentOptions(cfg)
	index := 0
	for i, option := range options {
		if option.Value == cfg.Agent {
			index = i
			break
		}
	}
	step := 1
	if previous {
		step = len(options) - 1
	}
	return options[(index+step)%len(options)].Value
}

// providerModelOptions lists the configured provider's models below the
// profiles. A failed lookup becomes one explanatory, unselectable-in-effect
// entry rather than an error, so the picker always opens.
func providerModelOptions(cfg core.Config) []tui.Option {
	if cfg.EffectiveProtocol() == core.ChatGPTProtocol {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	ids, err := provider.ListModels(ctx, cfg, &http.Client{})
	if err != nil {
		return []tui.Option{{Value: "@unavailable", Label: "Provider models unavailable", Description: err.Error()}}
	}
	options := make([]tui.Option, 0, len(ids))
	for _, id := range ids {
		description := "From the provider"
		if id == cfg.Model {
			description = "Current model"
		}
		options = append(options, tui.Option{Value: "@model:" + id, Label: id, Description: description})
	}
	return options
}
