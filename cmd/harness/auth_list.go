package main

import (
	"fmt"
	"io"
	"net/url"
	"sort"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tui"
)

// writeAuthList implements `goh auth list`, like `opencode auth list`: one row
// per provider route (the default, then each profile) naming where its
// credential comes from. No key, token or URL credential is ever printed.
func writeAuthList(w io.Writer, c core.Config) error {
	rows := [][3]string{}
	add := func(name string, cfg core.Config) {
		host := ""
		if endpoint, err := cfg.ProviderEndpoint(); err == nil {
			if u, err := url.Parse(endpoint); err == nil {
				host = u.Host
			}
		}
		if cfg.EffectiveProtocol() == core.ChatGPTProtocol {
			host = "chatgpt.com"
		}
		rows = append(rows, [3]string{name, host, cfg.CredentialSource()})
	}
	if c.Protocol != "" || c.Endpoint != "" || c.BaseURL != "" {
		add("default", c)
	}
	names := make([]string, 0, len(c.Profiles))
	for n := range c.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		cfg, err := c.Resolve(n, "")
		if err != nil {
			return err
		}
		add(n, cfg)
	}
	if c.EffectiveProtocol() != core.ChatGPTProtocol {
		chat := c
		chat.SetProviderRoute(core.ChatGPTProtocol, "", "")
		add("chatgpt", chat)
	}
	for _, r := range rows {
		fmt.Fprintf(w, "%-16s %-28s %s\n", tui.Safe(r[0]), tui.Safe(r[1]), tui.Safe(r[2]))
	}
	return nil
}
