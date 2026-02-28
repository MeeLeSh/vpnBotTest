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
	IsUsedTestPeriod   bool
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
	const query = `SELECT username, telegramId, accountDetailsLink, "isUsedTestPeriod" FROM "VpnUser" WHERE username = $1`

	row := db.QueryRow(ctx, query, username)

	var u VpnUser
	if err := row.Scan(&u.Username, &u.TelegramID, &u.AccountDetailsLink, &u.IsUsedTestPeriod); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

func createVpnUser(ctx context.Context, username string, telegramID int64, accountLink string, isUsedTestPeriod bool) error {
	const query = `INSERT INTO "VpnUser" (username, telegramId, accountDetailsLink, "isUsedTestPeriod") VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(ctx, query, username, telegramID, accountLink, isUsedTestPeriod)
	return err
}

func markTestPeriodUsed(ctx context.Context, username string) error {
	const query = `UPDATE "VpnUser" SET "isUsedTestPeriod" = TRUE WHERE username = $1`
	_, err := db.Exec(ctx, query, username)
	return err
}
