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
		return errors.New("usage: goh auth login|status|models|logout")
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
			if browser == "" || exec.CommandContext(ctx, browser, address).Run() != nil {
				fmt.Println("Open this OpenAI authorization URL in your browser:")
				fmt.Println(address)
				return nil
			}
			fmt.Println("Continue with ChatGPT in your browser. Waiting for authorization…")
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
		return errors.New("usage: goh auth login|status|models|logout")
	}
}
