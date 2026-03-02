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

// getQueuedInvoiceIDs returns all invoice_id values from "invoice_queue".
func getQueuedInvoiceIDs(ctx context.Context) ([]int64, error) {
	const query = `SELECT invoice_id FROM "invoice_queue"`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// insertInvoiceStatus saves a crypto invoice record to "invoice_queue" (id, invoice_id, telegramId, payload).
func insertInvoiceStatus(ctx context.Context, invoiceID int64, telegramID int64, payload string) error {
	const query = `INSERT INTO "invoice_queue" (invoice_id, "telegramId", payload) VALUES ($1, $2, $3)`
	_, err := db.Exec(ctx, query, invoiceID, telegramID, payload)
	return err
}

// deleteInvoiceStatusByInvoiceID removes the row from "invoice_queue" for the given invoice_id.
func deleteInvoiceStatusByInvoiceID(ctx context.Context, invoiceID int64) error {
	const query = `DELETE FROM "invoice_queue" WHERE invoice_id = $1`
	_, err := db.Exec(ctx, query, invoiceID)
	return err
}
