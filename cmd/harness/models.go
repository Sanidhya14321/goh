package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/provider"
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
	if *configPath == "" && *profile == "" {
		c = withChatGPTDefault(c)
	}
	if c.EffectiveProtocol() == core.ChatGPTProtocol {
		// Same list as goh auth models; the ChatGPT endpoint has no /models.
		return chatGPTAuth([]string{"models"})
	}
	endpoint, err := c.ProviderEndpoint()
	if err != nil {
		if errors.Is(err, core.ErrNoProvider) {
			return err
		}
		return fmt.Errorf("provider configuration: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ids, err := provider.ListModels(ctx, c, &http.Client{})
	if err != nil {
		return err
	}
	// Known providers also list models that cannot chat (speech, safety
	// classifiers, embeddings): show only the ones chat can use.
	p, _ := core.PresetForRoute(c.EffectiveProtocol(), c.BaseURL)
	var chat []string
	for _, id := range ids {
		if name, ok := p.ChatModel(id); ok {
			chat = append(chat, name)
		}
	}
	fmt.Fprintf(os.Stderr, "%d chat models from %s (current: %s)\n", len(chat), originOf(endpoint), c.Model)
	for _, id := range chat {
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
