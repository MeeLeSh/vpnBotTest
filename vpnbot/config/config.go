package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds environment-based configuration for the bot.
type Config struct {
	TelegramBotToken       string
	DatabaseURL            string
	CryptoPayToken         string
	CryptoPayTestnet       string
	CryptoPayAPIBaseURL    string
	CryptoPayAPITestnetURL string
	RemnawavePanelURL             string
	RemnawaveAPIToken              string
	RemnawaveSquadID  string // internal squad UUID to add when user subscribes (optional)
	Stars1Month                   int
	Stars3Months           int
	AdminTelegramID        int64
}

// AppConfig holds the loaded configuration for global access.
var AppConfig *Config

// LoadConfig reads required environment variables and returns a Config.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		TelegramBotToken:       os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		CryptoPayToken:         os.Getenv("CRYPTO_PAY_API_TOKEN"),
		CryptoPayTestnet:       os.Getenv("CRYPTO_PAY_TESTNET"),
		CryptoPayAPIBaseURL:    "https://pay.crypt.bot/api",
		CryptoPayAPITestnetURL: "https://testnet-pay.crypt.bot/api",
		RemnawavePanelURL:             os.Getenv("REMNAWAVE_PANEL_URL"),
		RemnawaveAPIToken:              os.Getenv("REMNAWAVE_API_TOKEN"),
		RemnawaveSquadID:  os.Getenv("REMNAWAVE_SQUAD_ID"),
		Stars1Month:                   1,
		Stars3Months:           3,
	}

	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is not set")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	if adminIDStr := os.Getenv("TELEGRAM_ADMIN_ID"); adminIDStr != "" {
		adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("TELEGRAM_ADMIN_ID is invalid: %w", err)
		}
		cfg.AdminTelegramID = adminID
	}

	return cfg, nil
}

