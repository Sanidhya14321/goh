package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/session"
)

// runConfigShow implements `goh config`, like `opencode debug config`: the
// effective configuration (defaults, user file, environment overrides and
// detected servers) as JSON. Values under env and headers are replaced, URL
// credentials and queries are dropped, and secret-looking text is masked.
func runConfigShow(args []string, w io.Writer) error {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	configPath := f.String("config", "", "explicit user JSON configuration")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("usage: goh config [--config FILE]")
	}
	c, err := core.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	c = withChatGPTDefault(c).WithDetectedLSP(nil).WithDetectedFormatters(nil)
	return writeConfig(w, c)
}

func writeConfig(w io.Writer, c core.Config) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	out, err := json.MarshalIndent(redactConfig(v, ""), "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, session.RedactSecrets(string(out)))
	return err
}

func redactConfig(v any, key string) any {
	switch x := v.(type) {
	case map[string]any:
		if key == "env" || key == "headers" {
			for k := range x {
				x[k] = "[redacted]"
			}
			return x
		}
		for k, child := range x {
			x[k] = redactConfig(child, strings.ToLower(k))
		}
		return x
	case []any:
		for i, child := range x {
			x[i] = redactConfig(child, key)
		}
		return x
	case string:
		if strings.Contains(x, "://") {
			return redactURL(x)
		}
		return x
	}
	return v
}
