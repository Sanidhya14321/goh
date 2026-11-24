package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/mcpclient"
	"github.com/Sanidhya14321/goh/internal/tui"
)

// runMCP implements `goh mcp list|login|logout`. list reports configuration
// only: no server is started, and secret values (env values, header values,
// URL credentials and queries) are never printed. login signs in to a remote
// server with OAuth in the browser; logout forgets that sign-in.
func runMCP(args []string) error {
	usage := errors.New("usage: goh mcp list [--config FILE] | goh mcp login NAME [--config FILE] | goh mcp logout NAME [--config FILE]")
	if len(args) == 0 || (args[0] != "list" && args[0] != "login" && args[0] != "logout") {
		return usage
	}
	f := flag.NewFlagSet("mcp "+args[0], flag.ContinueOnError)
	configPath := f.String("config", "", "explicit user JSON configuration")
	rest := args[1:]
	name := ""
	if args[0] != "list" && len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		name, rest = rest[0], rest[1:]
	}
	if err := f.Parse(rest); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 || (args[0] != "list" && name == "") {
		return usage
	}
	c, err := core.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	if args[0] == "list" {
		writeMCP(os.Stdout, c.MCP, os.LookupEnv)
		return nil
	}
	server, ok := c.MCP[name]
	if !ok {
		return fmt.Errorf("no MCP server named %q in the configuration (see goh mcp list)", name)
	}
	if server.URL == "" {
		return fmt.Errorf("MCP server %s is local (a command); only remote servers sign in", name)
	}
	if args[0] == "logout" {
		removed, err := mcpclient.Logout(server)
		if err != nil {
			return err
		}
		if !removed {
			fmt.Printf("Not signed in to %s.\n", tui.Safe(name))
			return nil
		}
		fmt.Printf("Signed out of %s.\n", tui.Safe(name))
		return nil
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, stop := context.WithTimeout(ctx, 10*time.Minute)
	defer stop()
	open := func(address string) error {
		fmt.Fprintf(os.Stderr, "Sign in to %s in your browser. If it did not open, visit:\n%s\n", tui.Safe(name), address)
		openBrowser(ctx, address)
		return nil
	}
	tools, err := mcpclient.Login(ctx, name, server, open)
	if err != nil {
		return err
	}
	fmt.Printf("Signed in to %s (%s).\n", tui.Safe(name), plural(tools, "tool", "tools"))
	return nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
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
		if mcpclient.UsesOAuth(s) {
			if _, err := core.ReadMCPCredential(s.URL); err == nil {
				fmt.Fprintln(w, "                 signed in")
			} else {
				fmt.Fprintln(w, "                 not signed in (goh mcp login "+tui.Safe(name)+" if it asks)")
			}
		}
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
