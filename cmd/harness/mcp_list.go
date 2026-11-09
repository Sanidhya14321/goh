package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tui"
)

// runMCP implements `goh mcp list`, like `opencode mcp list`. It reports
// configuration only: no server is started, and secret values (env values,
// header values, URL credentials and queries) are never printed.
func runMCP(args []string) error {
	usage := errors.New("usage: goh mcp list [--config FILE]")
	if len(args) == 0 || args[0] != "list" {
		return usage
	}
	f := flag.NewFlagSet("mcp list", flag.ContinueOnError)
	configPath := f.String("config", "", "explicit user JSON configuration")
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return usage
	}
	c, err := core.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	writeMCP(os.Stdout, c.MCP, os.LookupEnv)
	return nil
}

func writeMCP(w io.Writer, servers map[string]core.MCPConfig, lookup func(string) (string, bool)) {
	if len(servers) == 0 {
		fmt.Fprintln(w, "No MCP servers configured (add them under \"mcp\" in the config).")
		return
	}
	names := make([]string, 0, len(servers))
	for n := range servers {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		s := servers[name]
		kind, target := "local", strings.Join(append([]string{s.Command}, s.Args...), " ")
		if s.URL != "" {
			kind, target = "remote", redactURL(s.URL)
		}
		fmt.Fprintf(w, "%-16s %-6s %s\n", tui.Safe(name), kind, tui.Safe(target))
		var vars []string
		for k := range s.Env {
			vars = append(vars, "env "+k)
		}
		for h, env := range s.HeadersEnv {
			label := "header " + h + " from $" + env
			if _, ok := lookup(env); !ok {
				label += " (not set)"
			}
			vars = append(vars, label)
		}
		sort.Strings(vars)
		for _, v := range vars {
			fmt.Fprintln(w, "                 "+tui.Safe(v))
		}
	}
}

// redactURL drops credentials, query and fragment, which may hold tokens.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(invalid URL)"
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	return u.String()
}
