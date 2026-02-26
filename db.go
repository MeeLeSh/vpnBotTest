package main

import (
	"context"
	"errors"
	"os"
	"log"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

type VpnUser struct {
	TelegramID         int64
	AccountDetailsLink string
}

func initDB(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	log.Printf("DATABASE_URL: %q", dbURL)
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL environment variable is not set")
	}

	return pgxpool.New(ctx, dbURL)
}

func getVpnUserByTelegramID(ctx context.Context, telegramID int64) (*VpnUser, error) {
	const query = `SELECT telegramId, accountDetailsLink FROM "VpnUser" WHERE telegramId = $1`

	row := db.QueryRow(ctx, query, telegramID)

	var u VpnUser
	if err := row.Scan(&u.TelegramID, &u.AccountDetailsLink); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

func createVpnUser(ctx context.Context, telegramID int64, accountLink string) error {
	const query = `INSERT INTO "VpnUser" (telegramId, accountDetailsLink) VALUES ($1, $2)`
	_, err := db.Exec(ctx, query, telegramID, accountLink)
	return err
}

