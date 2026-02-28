package main

import (
	"context"
	"errors"
	"log"
	"os"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

type VpnUser struct {
	Username           string
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

func getVpnUserByUsername(ctx context.Context, username string) (*VpnUser, error) {
	const query = `SELECT username, telegramId, accountDetailsLink FROM "VpnUser" WHERE username = $1`

	row := db.QueryRow(ctx, query, username)

	var u VpnUser
	if err := row.Scan(&u.Username, &u.TelegramID, &u.AccountDetailsLink); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

func createVpnUser(ctx context.Context, username string, telegramID int64, accountLink string) error {
	const query = `INSERT INTO "VpnUser" (username, telegramId, accountDetailsLink) VALUES ($1, $2, $3)`
	_, err := db.Exec(ctx, query, username, telegramID, accountLink)
	return err
}

