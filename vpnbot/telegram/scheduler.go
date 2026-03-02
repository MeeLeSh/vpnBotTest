// Package telegram: scheduler that periodically fetches Crypto Pay invoices and sends Telegram messages when paid.

package telegram

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpnBot/vpnbot/api"
	"vpnBot/vpnbot/config"
	"vpnBot/vpnbot/db"
)

const schedulerInterval = 1 * time.Minute

// RunScheduler starts a loop that every minute calls getInvoices; for paid invoices
// sends the user a Telegram message "Subscription {plan} is paid" (plan and telegramId from payload).
// Does nothing if Crypto Pay is not configured (no token).
func RunScheduler(bot *tgbotapi.BotAPI) {
	if config.AppConfig == nil || config.AppConfig.CryptoPayToken == "" {
		log.Printf("Crypto Pay token for scheduler is not configured")
		return
	}

	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()

	checkInvoices(bot)

	for range ticker.C {
		checkInvoices(bot)
	}
}

// parsePayload extracts plan and telegram ID from payload.
// Supports: "plan:1month(tg_id=123,...)" or "plan:3months(tg_id=123,...)" or "plan:1month:tg:123".
func parsePayload(payload string) (plan string, telegramID int64, ok bool) {
	if payload == "" {
		return "", 0, false
	}
	// plan: from "plan:" until "(" or ":tg:" or end
	planPrefix := "plan:"
	if !strings.HasPrefix(payload, planPrefix) {
		return "", 0, false
	}
	rest := payload[len(planPrefix):]
	if i := strings.Index(rest, "("); i >= 0 {
		plan = strings.TrimSpace(rest[:i])
		// tg_id= from (tg_id=123,...)
		tgPart := rest[i+1:]
		if j := strings.Index(tgPart, "tg_id="); j >= 0 {
			tgPart = tgPart[j+len("tg_id="):]
			if k := strings.IndexAny(tgPart, ",)"); k >= 0 {
				tgPart = tgPart[:k]
			}
			id, err := strconv.ParseInt(strings.TrimSpace(tgPart), 10, 64)
			if err != nil {
				return "", 0, false
			}
			return plan, id, true
		}
		return "", 0, false
	}
	if i := strings.Index(rest, ":tg:"); i >= 0 {
		plan = strings.TrimSpace(rest[:i])
		tgPart := rest[i+len(":tg:"):]
		if k := strings.IndexAny(tgPart, " ,"); k >= 0 {
			tgPart = tgPart[:k]
		}
		id, err := strconv.ParseInt(strings.TrimSpace(tgPart), 10, 64)
		if err != nil {
			return "", 0, false
		}
		return plan, id, true
	}
	return strings.TrimSpace(rest), 0, false
}

func checkInvoices(bot *tgbotapi.BotAPI) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ids, err := db.GetQueuedInvoiceIDs(ctx)
	if err != nil {
		log.Printf("scheduler getQueuedInvoiceIDs: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	invoices, err := api.GetInvoices(ctx, ids)
	if err != nil {
		log.Printf("scheduler getInvoices: %v", err)
		return
	}

	for _, inv := range invoices {
		if inv.Status != "paid" {
			continue
		}
		plan, telegramID, ok := parsePayload(inv.Payload)
		if !ok {
			log.Printf("scheduler: could not parse payload for invoice %d: %q", inv.InvoiceID, inv.Payload)
			continue
		}

		// Apply subscription in Remnawave based on plan and Telegram ID.
		if err := api.Subscribe(ctx, telegramID, plan); err != nil {
			log.Printf("scheduler: subscribe %s failed for tg %d: %v", plan, telegramID, err)
			failMsg := tgbotapi.NewMessage(telegramID, "Payment received but activation failed: "+err.Error()+". Contact support.")
			if _, sendErr := bot.Send(failMsg); sendErr != nil {
				log.Printf("scheduler: send activation failed msg to %d: %v", telegramID, sendErr)
			}
			// keep invoice in queue for potential retry
			continue
		}

		// Notify user that subscription is applied.
		msg := tgbotapi.NewMessage(telegramID, "Subscription "+plan+" is paid")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("scheduler: send telegram to %d: %v", telegramID, err)
		}

		if err := db.DeleteInvoiceStatusByInvoiceID(ctx, inv.InvoiceID); err != nil {
			log.Printf("scheduler: delete invoice_queue for %d: %v", inv.InvoiceID, err)
		}
	}
}

