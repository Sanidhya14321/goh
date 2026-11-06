package main

import (
	"sort"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/tui"
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
