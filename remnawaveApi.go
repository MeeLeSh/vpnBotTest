package main

import (
	"fmt"
	"os"
)

// generateAccountLink builds a new account link for a given Telegram ID.
// The base URL is taken from ACCOUNT_BASE_URL, or falls back to a default.
func generateAccountLink(telegramID int64) string {
	baseURL := os.Getenv("ACCOUNT_BASE_URL")
	if baseURL == "" {
		baseURL = "https://your-domain.com/account"
	}

	return fmt.Sprintf("%s/new/%d", baseURL, telegramID)
}

