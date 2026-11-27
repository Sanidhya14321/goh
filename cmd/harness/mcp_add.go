package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tui"
)

type pairs map[string]string

func (p pairs) String() string { return "" }
func (p pairs) Set(v string) error {
	k, val, ok := strings.Cut(v, "=")
	if !ok || k == "" {
		return fmt.Errorf("%q must be NAME=VALUE", v)
	}
	p[k] = val
	return nil
}

// runMCPEdit implements `goh mcp add` and `goh mcp remove`, which edit the
// "mcp" section of the user settings. Nothing is started: the first chat
// or run that uses a server asks to start it.
func runMCPEdit(w io.Writer, args []string) error {
	verb := args[0]
	rest := args[1:]
	if len(rest) == 0 || strings.HasPrefix(rest[0], "-") {
		return fmt.Errorf("usage: goh mcp %s NAME …", verb)
	}
	name, rest := rest[0], rest[1:]
	path, raw, _, err := userSettings()
	if err != nil {
		return err
	}
	if verb == "remove" {
		if len(rest) != 0 {
			return errors.New("usage: goh mcp remove NAME")
		}
		next, err := core.RemoveMCPServer(raw, name)
		if err != nil {
			return err
		}
		if err := core.WriteUserConfig(path, next); err != nil {
			return err
		}
		fmt.Fprintf(w, "Removed MCP server %s.\n", tui.Safe(name))
		return nil
	}
	var command []string
	for i, a := range rest {
		if a == "--" {
			rest, command = rest[:i], rest[i+1:]
			break
		}
	}
	env, headers := pairs{}, pairs{}
	f := flag.NewFlagSet("mcp add", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	remote := f.String("url", "", "remote server URL (Streamable HTTP, or SSE with --sse)")
	sse := f.Bool("sse", false, "the server uses the older HTTP+SSE transport")
	client := f.String("oauth-client-id", "", "client registered with the server's authorization server")
	f.Var(env, "env", "NAME=VALUE for a local server's environment (repeatable)")
	f.Var(headers, "header", "HEADER=ENV_VAR: send HEADER with the value of ENV_VAR (repeatable)")
	if err := f.Parse(rest); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected %q; put the server command after --", f.Arg(0))
	}
	var server core.MCPConfig
	switch {
	case *remote != "" && len(command) > 0:
		return errors.New("give either --url or a command after --, not both")
	case *remote != "":
		if len(env) > 0 {
			return errors.New("--env is for local servers; a remote server takes --header NAME=ENV_VAR")
		}
		server = core.MCPConfig{URL: *remote, OAuthClientID: *client}
		if len(headers) > 0 {
			server.HeadersEnv = headers
		}
		if *sse {
			server.Transport = "sse"
		}
	case len(command) > 0:
		if len(headers) > 0 || *sse || *client != "" {
			return errors.New("--header, --sse and --oauth-client-id are for remote servers (--url)")
		}
		program := command[0]
		if !filepath.IsAbs(program) {
			// Stored as an absolute path, so a project's own folder or a
			// changed PATH cannot substitute another program later.
			if program, err = exec.LookPath(program); err != nil {
				return fmt.Errorf("%s is not installed or not on PATH", command[0])
			}
			if program, err = filepath.Abs(program); err != nil {
				return err
			}
		}
		server = core.MCPConfig{Command: program, Args: command[1:]}
		if len(env) > 0 {
			server.Env = env
		}
	default:
		return errors.New("give a command after -- (goh mcp add NAME -- COMMAND ARGS…) or --url URL")
	}
	next, err := core.SaveMCPServer(raw, name, server)
	if err != nil {
		return err
	}
	if err := core.WriteUserConfig(path, next); err != nil {
		return err
	}
	target := server.URL
	if target == "" {
		target = strings.Join(append([]string{server.Command}, server.Args...), " ")
	}
	fmt.Fprintf(w, "Added MCP server %s: %s\nIt starts, after your approval, in the next chat or run.\n", tui.Safe(name), tui.Safe(target))
	if len(server.Env) > 0 {
		fmt.Fprintln(w, "Its --env values are saved in the settings file; pass secrets to a remote server with --header NAME=ENV_VAR instead.")
	}
	return nil
}
