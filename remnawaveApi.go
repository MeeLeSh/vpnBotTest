package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	remapi "github.com/Jolymmiles/remnawave-api-go/v2/api"
)

// generateAccountLink builds a new account link for a given username and telegram ID.
// The base URL is taken from ACCOUNT_BASE_URL, or falls back to a default.
// It also creates the user in Remnawave with ExpireAt set to now().
func generateAccountLink(username string, telegramID int64) string {
	baseURL := os.Getenv("ACCOUNT_BASE_URL")
	if baseURL == "" {
		baseURL = "https://your-domain.com/account"
	}

	link := fmt.Sprintf("%s/new/%s", baseURL, username)

	panelURL := os.Getenv("REMNAWAVE_PANEL_URL")
	apiToken := os.Getenv("REMNAWAVE_API_TOKEN")

	if panelURL == "" || apiToken == "" {
		// If not configured, just skip the external call.
		return link
	}

	baseClient, err := remapi.NewClient(
		panelURL,
		remapi.StaticToken{Token: apiToken},
	)
	if err != nil {
		log.Printf("failed to create Remnawave base client: %v", err)
		return link
	}

	client := remapi.NewClientExt(baseClient)

	ctx := context.Background()

	_, err = client.Users().CreateUser(ctx, &remapi.CreateUserRequest{
		Username:   username,
		ExpireAt:   time.Now(),
		TelegramId: remapi.OptNilInt{Value: int(telegramID), Set: true},
	})
	if err != nil {
		log.Printf("failed to create Remnawave user %s: %v", username, err)
	}

	return link
}

