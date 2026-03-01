package main

import (
	"context"
	"fmt"
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
	}

	if getResp != nil {
		switch r := getResp.(type) {
		case *remapi.UserResponse:
			// User already exists, reuse its subscription URL
			return r.Response.SubscriptionUrl
		}
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

// Subscribe applies a subscription in Remnawave by extending the user's ExpireAt.
// Plan is one of: "1week_test", "1month", "3months".
func Subscribe(ctx context.Context, username string, telegramID int64, plan string) error {
	if remnawaveClient == nil {
		return nil
	}

	getResp, err := remnawaveClient.Users().GetUserByUsername(ctx, username)
	if err != nil {
		return err
	}

	var userResp *remapi.UserResponse
	switch r := getResp.(type) {
	case *remapi.UserResponse:
		userResp = r
	case *remapi.NotFoundError:
		return fmt.Errorf("user not found in panel: get your account link first (Profile)")
	default:
		return nil
	}

	userUUID := userResp.Response.UUID
	expireAt := userResp.Response.ExpireAt

	var extend time.Duration
	switch plan {
	case "1week_test":
		extend = 7 * 24 * time.Hour
	case "1month":
		extend = 30 * 24 * time.Hour
	case "3months":
		extend = 90 * 24 * time.Hour
	default:
		return nil
	}

	now := time.Now()
	var newExpireAt time.Time
	if expireAt.Before(now) || expireAt.Equal(now) {
		// Подписка истекла — прибавляем срок от сегодняшнего дня
		newExpireAt = now.Add(extend)
	} else {
		// Подписка ещё активна — прибавляем срок к текущей дате окончания
		newExpireAt = expireAt.Add(extend)
	}

	_, err = remnawaveClient.Users().UpdateUser(ctx, &remapi.UpdateUserRequest{
		UUID:     remapi.NewOptUUID(userUUID),
		ExpireAt: remapi.NewOptDateTime(newExpireAt),
	})
	return err
}
