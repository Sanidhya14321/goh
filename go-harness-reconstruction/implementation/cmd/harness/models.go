package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/provider"
)

// listModels prints the models offered by the configured provider (one per
// line on stdout), like `opencode models`. Configured profiles are listed on
// stderr so scripts can read the model IDs alone.
func listModels(args []string) error {
	f := flag.NewFlagSet("models", flag.ContinueOnError)
	configPath := f.String("config", "", "explicit user JSON configuration")
	profile := f.String("profile", "", "configured model profile to ask")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("usage: goh models [--profile NAME] [--config FILE]")
	}
	c, err := core.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	if c, err = c.Resolve(*profile, ""); err != nil {
		return err
	}
	endpoint, err := c.ProviderEndpoint()
	if err != nil {
		return fmt.Errorf("no provider configured (%w); run goh setup", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ids, err := provider.ListModels(ctx, c, &http.Client{})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%d models from %s (current: %s)\n", len(ids), originOf(endpoint), c.Model)
	for _, id := range ids {
		fmt.Println(id)
	}
	return nil
}

func originOf(endpoint string) string {
	u, err := provider.ModelsURL(endpoint)
	if err != nil {
		return "the configured provider"
	}
	// Only scheme and host; paths may carry provider credentials.
	if i := indexAfterHost(u); i > 0 {
		return u[:i]
	}
	return u
}

func indexAfterHost(u string) int {
	start := 0
	if i := len("https://"); len(u) > i && u[:i] == "https://" {
		start = i
	} else if i := len("http://"); len(u) > i && u[:i] == "http://" {
		start = i
	}
	for i := start; i < len(u); i++ {
		if u[i] == '/' {
			return i
		}
	}
	return len(u)
}
