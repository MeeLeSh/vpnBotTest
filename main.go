package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	if err := runMigrations(dbURL); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	var err error
	db, err = initDB(context.Background())
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer db.Close()

	if err := initRemnawaveClient(); err != nil {
		log.Fatalf("failed to init Remnawave client: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// Handle callback queries from inline buttons
		if update.CallbackQuery != nil {
			switch update.CallbackQuery.Data {
			case "sub1WeekTest":
				handleSub1WeekTest(bot, update.CallbackQuery)
			case "sub1Month":
				handleSub1Month(bot, update.CallbackQuery)
			case "sub3Months":
				handleSub3Months(bot, update.CallbackQuery)
			continue
			}
		}

		if update.Message == nil {
			continue
		}

		// Map button labels to commands so buttons can use friendly names
		switch update.Message.Text {
		case "Guide":
			update.Message.Text = "/instruction"
		case "Profile":
			update.Message.Text = "/account"
		case "Subscription":
			update.Message.Text = "/substribe"
		}

		if update.Message.IsCommand() || strings.HasPrefix(update.Message.Text, "/") {
			handleCommand(bot, update.Message)
			continue
		}

		handleMessage(bot, update.Message)
	}
}

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	cmd := msg.Command()
	if cmd == "" && strings.HasPrefix(msg.Text, "/") {
		cmd = strings.TrimPrefix(strings.Fields(msg.Text)[0], "/")
	}
	switch cmd {
	case "start":
		handleStartCommand(bot, msg)
	case "instruction":
		handleInstructionCommand(bot, msg)
	case "account":
		handleAccountCommand(bot, msg)
	case "substribe":
		handleSubstribeCommand(bot, msg)
	default:
		handleUnknownCommand(bot, msg)
	}
}

func handleStartCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := "Welcome! This bot helps you with VPN and account management.\n" +
		"For detailed step-by-step instructions, use the buttons below or type commands."

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Guide"),
			tgbotapi.NewKeyboardButton("Profile"),
			tgbotapi.NewKeyboardButton("Subscription"),
		),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = keyboard

	bot.Send(reply)
}

func handleInstructionCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := "Here is what you should do:\n" +
		"1. Step one description.\n" +
		"2. Step two description.\n" +
		"3. Step three description."
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.Send(reply)
}

func handleAccountCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	ctx := context.Background()
	telegramID := int64(msg.From.ID)

	username := msg.From.UserName
	if username == "" {
		username = fmt.Sprintf("tg-%d", telegramID)
	}

	user, err := getVpnUserByUsername(ctx, username)
	if err != nil {
		log.Printf("failed to get vpn user: %v", err)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Sorry, something went wrong. Please try again later.")
		bot.Send(reply)
		return
	}

	var link string

	if user == nil || user.AccountDetailsLink == "" {
		link = generateAccountLink(username, telegramID)
		
		if err := createVpnUser(ctx, username, telegramID, link, false); err != nil {
			log.Printf("failed to create vpn user: %v", err)
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Sorry, something went wrong while creating your account.")
			bot.Send(reply)
			return
		}
	} else {
		link = user.AccountDetailsLink
	}

	text := "Your account details are available here:\n" + link
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.Send(reply)
}

func handleSubstribeCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := "Choose a subscription option:"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 week test period", "sub1WeekTest"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 month (249 rub)", "sub1Month"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3 months (699 rub)", "sub3Months"),
		),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = keyboard

	bot.Send(reply)
}

func handleSub1WeekTest(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
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

	user, err := getVpnUserByUsername(ctx, username)
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
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "1 week test period is applied")
	bot.Send(msg)

	// mark test period as used
	if err := markTestPeriodUsed(ctx, username); err != nil {
		log.Printf("failed to mark test period used for %s: %v", username, err)
	}
}

func handleSub1Month(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	answer := tgbotapi.NewCallback(callback.ID, "")
	if _, err := bot.Request(answer); err != nil {
		log.Printf("failed to answer callback: %v", err)
	}

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "subscrubtion 1 is applied")
	bot.Send(msg)
}

func handleSub3Months(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	answer := tgbotapi.NewCallback(callback.ID, "")
	if _, err := bot.Request(answer); err != nil {
		log.Printf("failed to answer callback: %v", err)
	}

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "subscrubtion 3 is applied")
	bot.Send(msg)
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Unknown command. Try /start.")
	bot.Send(reply)
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, " ")
	bot.Send(reply)
}
