package db

import (
	"context"
	"errors"
	"log"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

type VpnUser struct {
	Username           string
	TelegramID         int64
	AccountDetailsLink string
	IsUsedTestPeriod   bool
}

// Init initializes the global database connection pool using the provided database URL.
// It stores the pool in a package-global variable for use by repository functions.
func Init(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	if dbURL == "" {
		return nil, errors.New("database url is empty")
	}
	log.Printf("DATABASE_URL: %q", dbURL)
	p, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	pool = p
	return p, nil
}

// Close closes the global database connection pool if it was initialized.
func Close() {
	if pool != nil {
		pool.Close()
	}
}

func GetVpnUserByUsername(ctx context.Context, username string) (*VpnUser, error) {
	const query = `SELECT username, telegramId, accountDetailsLink, "isUsedTestPeriod" FROM "VpnUser" WHERE username = $1`

	row := pool.QueryRow(ctx, query, username)

	var u VpnUser
	if err := row.Scan(&u.Username, &u.TelegramID, &u.AccountDetailsLink, &u.IsUsedTestPeriod); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

func CreateVpnUser(ctx context.Context, username string, telegramID int64, accountLink string, isUsedTestPeriod bool) error {
	const query = `INSERT INTO "VpnUser" (username, telegramId, accountDetailsLink, "isUsedTestPeriod") VALUES ($1, $2, $3, $4)`
	_, err := pool.Exec(ctx, query, username, telegramID, accountLink, isUsedTestPeriod)
	return err
}

func MarkTestPeriodUsed(ctx context.Context, username string) error {
	const query = `UPDATE "VpnUser" SET "isUsedTestPeriod" = TRUE WHERE username = $1`
	_, err := pool.Exec(ctx, query, username)
	return err
}

// GetQueuedInvoiceIDs returns all invoice_id values from "invoice_queue".
func GetQueuedInvoiceIDs(ctx context.Context) ([]int64, error) {
	const query = `SELECT invoice_id FROM "invoice_queue"`

	rows, err := pool.Query(ctx, query)
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

// InsertInvoiceStatus saves a crypto invoice record to "invoice_queue" (id, invoice_id, telegramId, payload).
func InsertInvoiceStatus(ctx context.Context, invoiceID int64, telegramID int64, payload string) error {
	const query = `INSERT INTO "invoice_queue" (invoice_id, "telegramId", payload) VALUES ($1, $2, $3)`
	_, err := pool.Exec(ctx, query, invoiceID, telegramID, payload)
	return err
}

// DeleteInvoiceStatusByInvoiceID removes the row from "invoice_queue" for the given invoice_id.
func DeleteInvoiceStatusByInvoiceID(ctx context.Context, invoiceID int64) error {
	const query = `DELETE FROM "invoice_queue" WHERE invoice_id = $1`
	_, err := pool.Exec(ctx, query, invoiceID)
	return err
}

