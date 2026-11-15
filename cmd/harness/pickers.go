package main

import (
	"sort"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tui"
)

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
