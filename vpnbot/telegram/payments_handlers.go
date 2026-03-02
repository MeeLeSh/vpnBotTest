package telegram

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpnBot/vpnbot/api"
	"vpnBot/vpnbot/config"
	"vpnBot/vpnbot/db"
)

// HandlePlanChoice shows payment method buttons (Stars / Crypto) for the chosen plan.
func HandlePlanChoice(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, plan string) {
	answer := tgbotapi.NewCallback(callback.ID, "")
	if _, err := bot.Request(answer); err != nil {
		log.Printf("failed to answer callback: %v", err)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Pay Telegram Stars", "pay_stars_"+plan),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Pay crypto", "pay_crypto_"+plan),
		),
	)
	reply := tgbotapi.NewMessage(callback.Message.Chat.ID, "Choose payment method:")
	reply.ReplyMarkup = keyboard
	bot.Send(reply)
}

func HandlePayCryptoPlan1Month(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	handlePayCryptoPlan(bot, callback,
		"249",
		"VPN subscription — 1 month",
		"1month",
		"Pay for 1 month subscription (249 ₽):\n",
	)
}

func HandlePayCryptoPlan3Months(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	handlePayCryptoPlan(bot, callback,
		"699",
		"VPN subscription — 3 months",
		"3months",
		"Pay for 3 months subscription (699 ₽):\n",
	)
}

// handlePayCryptoPlan contains common logic for creating and sending a Crypto Pay invoice
// and storing it in the invoice_queue.
func handlePayCryptoPlan(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, amount, description, plan, prefixText string) {
	answer := tgbotapi.NewCallback(callback.ID, "")
	if _, err := bot.Request(answer); err != nil {
		log.Printf("failed to answer callback: %v", err)
	}

	inv, err := api.CreateInvoice(context.Background(), api.CreateInvoiceOpts{
		CurrencyType:  "fiat",
		Fiat:          "RUB",
		Amount:        amount,
		Description:   description,
		HiddenMessage: "Subscription is paid successfully",
		Payload:       fmt.Sprintf("plan:%s(tg_id=%d,username=%s)", plan, callback.From.ID, callback.From.UserName),
	})

	if err != nil {
		log.Printf("crypto createInvoice: %v", err)
		bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "Failed to create crypto invoice. Please try again later."))
		return
	}
	if inv == nil {
		bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "Crypto payment is not configured. Contact support."))
		return
	}

	ctx := context.Background()
	if err := db.InsertInvoiceStatus(ctx, inv.InvoiceID, int64(callback.From.ID), inv.Payload); err != nil {
		log.Printf("insertInvoiceStatus: %v", err)
	}

	text := prefixText + inv.BotInvoiceURL
	bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, text))
}

func HandleSub1WeekTest(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	answer := tgbotapi.NewCallback(callback.ID, "")
	if _, err := bot.Request(answer); err != nil {
		log.Printf("failed to answer callback: %v", err)
	}

	ctx := context.Background()
	telegramID := int64(callback.From.ID)
	username := callback.From.UserName
	if username == "" {
		username = fmt.Sprintf("tg-%d", telegramID)
	}

	user, err := db.GetVpnUserByUsername(ctx, username)
	if err != nil {
		log.Printf("failed to get vpn user for test period: %v", err)
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Something went wrong. Please try again later.")
		bot.Send(msg)
		return
	}

	if user != nil && user.IsUsedTestPeriod {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "you already used test period")
		bot.Send(msg)
		return
	}

	// apply test period
	if err := api.Subscribe(ctx, telegramID, "1week_test"); err != nil {
		log.Printf("subscribe 1week_test failed for %s: %v", username, err)
		bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "Failed: "+err.Error()))
		return
	}
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "1 week test period is applied")
	bot.Send(msg)

	// mark test period as used
	if err := db.MarkTestPeriodUsed(ctx, username); err != nil {
		log.Printf("failed to mark test period used for %s: %v", username, err)
	}
}

func HandlePayStarsPlan1Month(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	answer := tgbotapi.NewCallback(callback.ID, "")
	if _, err := bot.Request(answer); err != nil {
		log.Printf("failed to answer callback: %v", err)
	}
	sendStarsInvoice(bot, callback.Message.Chat.ID, "1 month", "VPN subscription for 1 month", "1 month", "plan:1month", config.AppConfig.Stars1Month)
}

func HandlePayStarsPlan3Months(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	answer := tgbotapi.NewCallback(callback.ID, "")
	if _, err := bot.Request(answer); err != nil {
		log.Printf("failed to answer callback: %v", err)
	}
	sendStarsInvoice(bot, callback.Message.Chat.ID, "3 months", "VPN subscription for 3 months", "3 months", "plan:3months", config.AppConfig.Stars3Months)
}

// sendStarsInvoice sends a Telegram Stars (XTR) invoice. providerToken empty for digital goods.
// priceLineLabel: label for the single price line (avoid "Invoice"); description: product description only.
func sendStarsInvoice(bot *tgbotapi.BotAPI, chatID int64, title, description, priceLineLabel, payload string, starsAmount int) {
	prices := []tgbotapi.LabeledPrice{{Label: priceLineLabel, Amount: starsAmount}}
	invoice := tgbotapi.NewInvoice(chatID, title, description, payload, "", "", "XTR", prices)
	invoice.SuggestedTipAmounts = []int{} // required for Stars: must be an array (empty = no tips)
	if _, err := bot.Send(invoice); err != nil {
		log.Printf("sendInvoice failed: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Failed to create payment. Please try again later."))
	}
}

func HandlePreCheckoutQuery(bot *tgbotapi.BotAPI, query *tgbotapi.PreCheckoutQuery) {
	// Accept any payload we issued (plan:1month, plan:3months). Optionally validate stock/price here.
	cfg := tgbotapi.PreCheckoutConfig{PreCheckoutQueryID: query.ID, OK: true}
	if _, err := bot.Request(cfg); err != nil {
		log.Printf("answerPreCheckoutQuery failed: %v", err)
	}
}

func HandleSuccessfulStarsPayment(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	payload := msg.SuccessfulPayment.InvoicePayload
	// payload is e.g. "plan:1month" or "plan:3months"
	var plan string
	switch payload {
	case "plan:1month":
		plan = "1month"
	case "plan:3months":
		plan = "3months"
	default:
		log.Printf("unknown payment payload: %s", payload)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Unknown plan. Contact support."))
		return
	}

	ctx := context.Background()
	telegramID := int64(msg.From.ID)
	username := msg.From.UserName
	if username == "" {
		username = fmt.Sprintf("tg-%d", telegramID)
	}

	if err := api.Subscribe(ctx, telegramID, plan); err != nil {
		log.Printf("subscribe %s after payment failed for %s: %v", plan, username, err)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Payment received but activation failed: "+err.Error()+". Contact support."))
		return
	}

	text := "Payment successful. Subscription is active."
	if plan == "1month" {
		text = "Subscription 1 month is applied."
	} else {
		text = "Subscription 3 months is applied."
	}
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}

