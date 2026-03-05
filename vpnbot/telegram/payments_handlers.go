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
			tgbotapi.NewInlineKeyboardButtonData("Оплатить Telegram Stars", "pay_stars_"+plan),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Оплатить криптовалютой", "pay_crypto_"+plan),
		),
	)
	reply := tgbotapi.NewMessage(callback.Message.Chat.ID, "Выберите способ оплаты:")
	reply.ReplyMarkup = keyboard
	bot.Send(reply)
}

func HandlePayCryptoPlan1Month(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	handlePayCryptoPlan(bot, callback,
		"249",
		"Подписка VPN — 1 месяц",
		"1month",
		"Оплатите подписку на 1 месяц (249 ₽):\n",
	)
}

func HandlePayCryptoPlan3Months(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	handlePayCryptoPlan(bot, callback,
		"699",
		"Подписка VPN — 3 месяца",
		"3months",
		"Оплатите подписку на 3 месяца (699 ₽):\n",
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
		HiddenMessage: "Подписка успешно оплачена",
		Payload:       fmt.Sprintf("plan:%s(tg_id=%d,username=%s)", plan, callback.From.ID, callback.From.UserName),
	})

	if err != nil {
		log.Printf("crypto createInvoice: %v", err)
		bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "Не удалось создать крипто‑счёт. Попробуйте позже."))
		return
	}
	if inv == nil {
		bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "Оплата криптовалютой не настроена. Свяжитесь с поддержкой."))
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
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Произошла ошибка. Пожалуйста, попробуйте позже.")
		bot.Send(msg)
		return
	}

	if user != nil && user.IsUsedTestPeriod {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Вы уже использовали тестовый период.")
		bot.Send(msg)
		return
	}

	// apply test period
	if err := api.Subscribe(ctx, telegramID, "1week_test"); err != nil {
		log.Printf("subscribe 1week_test failed for %s: %v", username, err)
		bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "Ошибка активации тестового периода: "+err.Error()))
		return
	}
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "Тестовый период на 1 неделю активирован.")
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
	sendStarsInvoice(bot, callback.Message.Chat.ID, "Подписка на 1 месяц", "Подписка VPN на 1 месяц", "1 месяц", "plan:1month", config.AppConfig.Stars1Month)
}

func HandlePayStarsPlan3Months(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	answer := tgbotapi.NewCallback(callback.ID, "")
	if _, err := bot.Request(answer); err != nil {
		log.Printf("failed to answer callback: %v", err)
	}
	sendStarsInvoice(bot, callback.Message.Chat.ID, "Подписка на 3 месяца", "Подписка VPN на 3 месяца", "3 месяца", "plan:3months", config.AppConfig.Stars3Months)
}

// sendStarsInvoice sends a Telegram Stars (XTR) invoice. providerToken empty for digital goods.
// priceLineLabel: label for the single price line (avoid "Invoice"); description: product description only.
func sendStarsInvoice(bot *tgbotapi.BotAPI, chatID int64, title, description, priceLineLabel, payload string, starsAmount int) {
	prices := []tgbotapi.LabeledPrice{{Label: priceLineLabel, Amount: starsAmount}}
	invoice := tgbotapi.NewInvoice(chatID, title, description, payload, "", "", "XTR", prices)
	invoice.SuggestedTipAmounts = []int{} // required for Stars: must be an array (empty = no tips)
	if _, err := bot.Send(invoice); err != nil {
		log.Printf("sendInvoice failed: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Не удалось создать платёж. Пожалуйста, попробуйте позже."))
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
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Неизвестный тариф. Свяжитесь с поддержкой."))
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
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Платёж получен, но не удалось активировать подписку: "+err.Error()+". Свяжитесь с поддержкой."))
		return
	}

	var text string
	if plan == "1month" {
		text = "Подписка на 1 месяц активирована."
	} else if plan == "3months" {
		text = "Подписка на 3 месяца активирована."
	} else {
		text = "Подписка активирована."
	}
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}
