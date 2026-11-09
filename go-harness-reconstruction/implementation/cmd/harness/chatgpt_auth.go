package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	"example.com/independent-harness/internal/core"
	"example.com/independent-harness/internal/tui"
)

func chatGPTAuth(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: goh auth login|status|models|logout|list")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	switch args[0] {
	case "login":
		model, err := core.LoginChatGPT(ctx, func(address string) error {
			var browser string
			switch runtime.GOOS {
			case "darwin":
				browser = "open"
			case "linux":
				browser = "xdg-open"
			}
			// Always show the link, so another browser or a private window can
			// be used to sign in to a different account.
			fmt.Println("Open this OpenAI authorization URL in your browser:")
			fmt.Println(address)
			if browser != "" && exec.CommandContext(ctx, browser, address).Run() == nil {
				fmt.Println("(Also opened in your default browser.)")
			}
			fmt.Println("Waiting for authorization…")
			return nil
		})
		if err != nil {
			return err
		}
		fmt.Println("ChatGPT authorization saved.")
		if model != "" {
			fmt.Println("Default model:", tui.Safe(model))
		}
		return nil
	case "status":
		model, err := core.ChatGPTModel()
		if err != nil {
			return err
		}
		fmt.Println("ChatGPT authorization is saved locally.")
		if model != "" {
			fmt.Println("Default model:", tui.Safe(model))
		}
		return nil
	case "list":
		c, err := core.LoadConfig("")
		if err != nil {
			return err
		}
		return writeAuthList(os.Stdout, c)
	case "models":
		models, err := core.ListChatGPTModels(ctx)
		if err != nil {
			return err
		}
		for _, model := range models {
			fmt.Printf("%s  %s\n", tui.Safe(model.Slug), tui.Safe(model.Name))
		}
		return nil
	case "logout":
		if err := core.LogoutChatGPT(ctx); err != nil {
			return err
		}
		fmt.Println("ChatGPT authorization cleared.")
		return nil
	default:
		return errors.New("usage: goh auth login|status|models|logout|list")
	}
}
