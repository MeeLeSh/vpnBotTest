package api

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	remapi "github.com/Jolymmiles/remnawave-api-go/v2/api"

	"vpnBot/vpnbot/config"
)

var remnawaveClient *remapi.ClientExt

// InitRemnawaveClient initializes the Remnawave API client using AppConfig.
func InitRemnawaveClient() error {
	if config.AppConfig == nil || config.AppConfig.RemnawavePanelURL == "" || config.AppConfig.RemnawaveAPIToken == "" {
		return nil
	}

	baseClient, err := remapi.NewClient(
		config.AppConfig.RemnawavePanelURL,
		remapi.StaticToken{Token: config.AppConfig.RemnawaveAPIToken},
	)
	if err != nil {
		return err
	}

	remnawaveClient = remapi.NewClientExt(baseClient)
	return nil
}

// GenerateAccountLink creates (or reuses) a user in Remnawave and returns its SubscriptionUrl.
// It first looks up the user by Telegram ID, then falls back to creating a new user.
func GenerateAccountLink(username string, telegramID int64) string {
	if remnawaveClient == nil {
		log.Printf("Remnawave client not initialized (REMNAWAVE_PANEL_URL/REMNAWAVE_API_TOKEN not set)")
		return ""
	}

	ctx := context.Background()

	// First, try to get existing user by Telegram ID
	getResp, err := remnawaveClient.Users().GetUserByTelegramId(ctx, strconv.FormatInt(telegramID, 10))
	if err != nil {
		log.Printf("failed to call Remnawave GetUserByTelegramId %d: %v", telegramID, err)
	} else if getResp != nil {
		if r, ok := getResp.(*remapi.UsersResponse); ok {
			if len(r.Response) > 0 {
				// User already exists, reuse its subscription URL
				return r.Response[0].SubscriptionUrl
			}
		} else {
			log.Printf("unexpected GetUserByTelegramId response type: %T", getResp)
		}
	}

	// Not found or lookup failed → create a new user
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
// User is resolved by Telegram ID via Remnawave Users API.
func Subscribe(ctx context.Context, telegramID int64, plan string) error {
	if remnawaveClient == nil {
		return nil
	}

	// API expects Telegram ID as string.
	getResp, err := remnawaveClient.Users().GetUserByTelegramId(ctx, strconv.FormatInt(telegramID, 10))
	if err != nil {
		return err
	}

	var userUUID uuid.UUID
	var expireAt time.Time

	switch r := getResp.(type) {
	case *remapi.UsersResponse:
		if len(r.Response) == 0 {
			return fmt.Errorf("user not found in panel: get your account link first (Profile)")
		}
		item := r.Response[0]
		userUUID = item.UUID
		expireAt = item.ExpireAt
	default:
		return fmt.Errorf("unexpected GetUserByTelegramId response type: %T", r)
	}

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

	// Set a single squad from config when configured.
	var activeSquads []uuid.UUID
	if config.AppConfig != nil && config.AppConfig.RemnawaveSquadID != "" {
		squadUUID, err := uuid.Parse(config.AppConfig.RemnawaveSquadID)
		if err != nil {
			log.Printf("invalid REMNAWAVE_SQUAD_ID UUID %q: %v", config.AppConfig.RemnawaveSquadID, err)
		} else {
			activeSquads = []uuid.UUID{squadUUID}
		}
	}

	req := &remapi.UpdateUserRequest{
		UUID:            remapi.NewOptUUID(userUUID),
		ExpireAt:        remapi.NewOptDateTime(newExpireAt),
		HwidDeviceLimit: remapi.NewOptNilInt(3),
	}
	if len(activeSquads) > 0 {
		req.ActiveInternalSquads = activeSquads
	}
	_, err = remnawaveClient.Users().UpdateUser(ctx, req)
	return err
}

