package db

import (
	"context"
	"errors"
	"log"
	"time"

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

// UserQuestion represents a user question sent after pressing Help.
type UserQuestion struct {
	ID          int64
	TelegramID  int64
	Message     string
	MessageDate time.Time
	Answer      string // admin reply; empty until answered
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

// InsertUserQuestion saves a user question to "UserQuestion" (telegramId, message, messageDate).
func InsertUserQuestion(ctx context.Context, telegramID int64, message string, messageDate time.Time) error {
	const query = `INSERT INTO "UserQuestion" ("telegramId", message, "messageDate") VALUES ($1, $2, $3)`
	_, err := pool.Exec(ctx, query, telegramID, message, messageDate)
	return err
}

// GetAllUserQuestions returns all stored user questions ordered by most recent first.
func GetAllUserQuestions(ctx context.Context) ([]UserQuestion, error) {
	const query = `SELECT id, "telegramId", message, "messageDate", COALESCE(answer, '') FROM "UserQuestion" ORDER BY "messageDate" DESC`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var qs []UserQuestion
	for rows.Next() {
		var q UserQuestion
		if err := rows.Scan(&q.ID, &q.TelegramID, &q.Message, &q.MessageDate, &q.Answer); err != nil {
			return nil, err
		}
		qs = append(qs, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return qs, nil
}

// GetUnansweredUserQuestions returns questions where answer is null or empty, ordered by most recent first.
func GetUnansweredUserQuestions(ctx context.Context) ([]UserQuestion, error) {
	const query = `SELECT id, "telegramId", message, "messageDate", COALESCE(answer, '') FROM "UserQuestion" WHERE (answer IS NULL OR TRIM(answer) = '') ORDER BY "messageDate" DESC`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var qs []UserQuestion
	for rows.Next() {
		var q UserQuestion
		if err := rows.Scan(&q.ID, &q.TelegramID, &q.Message, &q.MessageDate, &q.Answer); err != nil {
			return nil, err
		}
		qs = append(qs, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return qs, nil
}

// GetUserQuestionByID returns a single user question by its ID.
func GetUserQuestionByID(ctx context.Context, id int64) (*UserQuestion, error) {
	const query = `SELECT id, "telegramId", message, "messageDate", COALESCE(answer, '') FROM "UserQuestion" WHERE id = $1`
	row := pool.QueryRow(ctx, query, id)
	var q UserQuestion
	if err := row.Scan(&q.ID, &q.TelegramID, &q.Message, &q.MessageDate, &q.Answer); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &q, nil
}

// UpdateUserQuestionAnswer sets the answer for the given question ID.
func UpdateUserQuestionAnswer(ctx context.Context, questionID int64, answer string) error {
	const query = `UPDATE "UserQuestion" SET answer = $1 WHERE id = $2`
	_, err := pool.Exec(ctx, query, answer, questionID)
	return err
}

