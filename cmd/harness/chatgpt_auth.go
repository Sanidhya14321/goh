package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/Sanidhya14321/goh/internal/core"
	"github.com/Sanidhya14321/goh/internal/tui"
)

const authUsage = "usage: goh auth login|switch|status|models|logout|list|store [file|keychain]"

func chatGPTAuth(args []string) error {
	if len(args) == 2 && args[0] == "store" || len(args) == 1 && args[0] == "store" {
		return authStore(args[1:])
	}
	if len(args) != 1 {
		return errors.New(authUsage)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	switch args[0] {
	case "login", "switch":
		switching := args[0] == "switch"
		reauthorizing := !switching && core.ChatGPTSignedIn()
		login := core.LoginChatGPT
		if switching {
			login = core.SwitchChatGPTAccount
		}
		model, err := login(ctx, func(address string) error {
			// Always show the link, so another browser or a private window can
			// be used to sign in to a different account.
			fmt.Println("Open this OpenAI authorization URL in your browser:")
			fmt.Println(address)
			if openBrowser(ctx, address) {
				fmt.Println("(Also opened in your default browser.)")
			}
			if reauthorizing {
				fmt.Println("Sign in with the same ChatGPT account and workspace as before. To use another one, press Ctrl+C and run `goh auth switch` (or /connect → Use a different ChatGPT account).")
			}
			fmt.Println("Waiting for authorization… If OpenAI shows an error page instead, press Ctrl+C here.")
			return nil
		})
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			hint := "ChatGPT sign-in was not completed."
			if reauthorizing {
				hint += " If OpenAI said the app is only available to members of its workspace, the browser is signed in to a different ChatGPT account or workspace: run `goh auth switch` to sign in with that one instead."
			}
			return errors.New(hint)
		}
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
		if where := core.ChatGPTKeptIn(); where != "" {
			fmt.Println("ChatGPT authorization is saved in the " + where + ".")
		} else {
			fmt.Println("ChatGPT authorization is saved locally.")
		}
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
		return errors.New(authUsage)
	}
}

// authStore shows or changes where saved credentials are kept.
func authStore(args []string) error {
	if len(args) == 0 {
		store, err := core.CredentialStore()
		if err != nil {
			return err
		}
		if store == core.StoreKeychain {
			fmt.Println("Saved credentials are kept in the " + core.CredentialStoreName() + ".")
		} else {
			fmt.Println("Saved credentials are kept in private files (OS store: " + core.CredentialStoreName() + ").")
		}
		return nil
	}
	moved, err := core.SetCredentialStore(args[0])
	if err != nil {
		return err
	}
	where := "private files"
	if args[0] == core.StoreKeychain {
		where = "the " + core.CredentialStoreName()
	}
	fmt.Printf("Saved credentials are now kept in %s (%d moved).\n", where, moved)
	return nil
}
