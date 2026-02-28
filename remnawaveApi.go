package main

import (
	"context"
	"log"
	"os"
	"time"

	remapi "github.com/Jolymmiles/remnawave-api-go/v2/api"
)

var remnawaveClient *remapi.ClientExt

func initRemnawaveClient() error {
	panelURL := os.Getenv("REMNAWAVE_PANEL_URL")
	apiToken := os.Getenv("REMNAWAVE_API_TOKEN")

	if panelURL == "" || apiToken == "" {
		return nil
	}

	baseClient, err := remapi.NewClient(
		panelURL,
		remapi.StaticToken{Token: apiToken},
	)
	if err != nil {
		return err
	}

	remnawaveClient = remapi.NewClientExt(baseClient)
	return nil
}

// generateAccountLink creates a user in Remnawave and returns its SubscriptionUrl.
func generateAccountLink(username string, telegramID int64) string {
	if remnawaveClient == nil {
		log.Printf("Remnawave client not initialized (REMNAWAVE_PANEL_URL/REMNAWAVE_API_TOKEN not set)")
		return ""
	}

	ctx := context.Background()

	// First, try to get existing user by username
	getResp, err := remnawaveClient.Users().GetUserByUsername(ctx, username)
	if err != nil {
		log.Printf("failed to call Remnawave GetUserByUsername %s: %v", username, err)
		return ""
	}

	switch r := getResp.(type) {
	case *remapi.UserResponse:
		// User already exists, reuse its subscription URL
		return r.Response.SubscriptionUrl
	case *remapi.NotFoundError:
		// User does not exist, fall through to create
	default:
		log.Printf("unexpected GetUserByUsername response type: %T", getResp)
		return ""
	}

	// Not found → create a new user
	createResp, err := remnawaveClient.Users().CreateUser(ctx, &remapi.CreateUserRequest{
		Username:   username,
		ExpireAt:   time.Now(),
		TelegramId: remapi.OptNilInt{Value: int(telegramID), Set: true},
	})
	if err != nil {
		log.Printf("failed to create Remnawave user %s: %v", username, err)
		return ""
	}

	if created, ok := createResp.(*remapi.UserResponse); ok {
		return created.Response.SubscriptionUrl
	}

	log.Printf("unexpected CreateUser response type: %T", createResp)
	return ""
}

