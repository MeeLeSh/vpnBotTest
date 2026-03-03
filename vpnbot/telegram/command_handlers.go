package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpnBot/vpnbot/api"
	"vpnBot/vpnbot/config"
	"vpnBot/vpnbot/db"
)

func HandleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
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
	case "help":
		handleHelpCommand(bot, msg)
	case "questions":
		handleQuestionsCommand(bot, msg)
	default:
		handleUnknownCommand(bot, msg)
	}
}

func handleStartCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	_, _, _, link, ok := resolveUserAndLink(bot, msg)
	if !ok {
		return
	}
	_ = link // link ensured for reply keyboard / account state

	text := "Welcome! This bot helps you with VPN and account management.\n" +
		"For detailed step-by-step instructions, use the buttons below or type commands."

	row1 := tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Guide"),
		tgbotapi.NewKeyboardButton("Profile"),
	)

	// Second row always has Subscription and Help.
	row2Buttons := []tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton("Subscription"),
		tgbotapi.NewKeyboardButton("Help"),
	}
	// If this user is the admin, add Questions button near Help.
	if config.AppConfig != nil && config.AppConfig.AdminTelegramID != 0 && int64(msg.From.ID) == config.AppConfig.AdminTelegramID {
		row2Buttons = append(row2Buttons, tgbotapi.NewKeyboardButton("Questions"))
	}
	row2 := tgbotapi.NewKeyboardButtonRow(row2Buttons...)

	keyboard := tgbotapi.NewReplyKeyboard(row1, row2)

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
	_, _, _, link, ok := resolveUserAndLink(bot, msg)
	if !ok {
		return
	}

	text := "Your account details are available here:\n" + link
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.Send(reply)
}

func handleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	SetAwaitingHelpQuestion(msg.Chat.ID)
	reply := tgbotapi.NewMessage(msg.Chat.ID, "write your question to the chat")
	bot.Send(reply)
}

func handleQuestionsCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Only bot admin is allowed to see questions.
	if config.AppConfig == nil || config.AppConfig.AdminTelegramID == 0 || int64(msg.From.ID) != config.AppConfig.AdminTelegramID {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "This command is only available to the bot admin.")
		bot.Send(reply)
		return
	}

	ctx := context.Background()
	questions, err := db.GetAllUserQuestions(ctx)
	if err != nil {
		log.Printf("failed to load user questions: %v", err)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Failed to load questions. Please try again later.")
		bot.Send(reply)
		return
	}

	if len(questions) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "There are no questions yet.")
		bot.Send(reply)
		return
	}

	for _, q := range questions {
		text := fmt.Sprintf("Question:\n%s\n\nTelegram ID: %d", q.Message, q.TelegramID)

		// Button to open a chat with the user to answer the question.
		answerBtn := tgbotapi.NewInlineKeyboardButtonURL(
			"Answer",
			fmt.Sprintf("tg://user?id=%d", q.TelegramID),
		)
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(answerBtn),
		)

		out := tgbotapi.NewMessage(msg.Chat.ID, text)
		out.ReplyMarkup = markup
		bot.Send(out)
	}
}

func handleSubstribeCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := "Choose a subscription plan:"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 week test period", "plan_1week"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 month (249 rub)", "plan_1month"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3 months (699 rub)", "plan_3months"),
		),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = keyboard

	bot.Send(reply)
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Unknown command. Try /start.")
	bot.Send(reply)
}

func HandleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, " ")
	bot.Send(reply)
}

// HandleState processes non-command messages, including help questions state.
func HandleState(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// If user was prompted to write a question (after Help), save it to UserQuestion
	if IsAwaitingHelpQuestion(msg.Chat.ID) && msg.Text != "" {
		ClearAwaitingHelpQuestion(msg.Chat.ID)
		msgTime := time.Unix(int64(msg.Date), 0)
		if err := db.InsertUserQuestion(context.Background(), int64(msg.From.ID), msg.Text, msgTime); err != nil {
			log.Printf("failed to save user question: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Sorry, we couldn't save your question. Please try again later."))
		} else {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Your question has been saved. We'll get back to you soon."))
		}
		return
	}

	HandleMessage(bot, msg)
}

// resolveUserAndLink resolves username from msg, loads or creates VPN user, and returns account link.
// On failure it sends an error message to the chat and returns ok == false.
func resolveUserAndLink(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) (ctx context.Context, telegramID int64, username, link string, ok bool) {
	ctx = context.Background()
	telegramID = int64(msg.From.ID)
	username = msg.From.UserName
	if username == "" {
		username = fmt.Sprintf("tg-%d", telegramID)
	}

	user, err := db.GetVpnUserByUsername(ctx, username)
	if err != nil {
		log.Printf("failed to get vpn user: %v", err)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Sorry, something went wrong. Please try again later."))
		return ctx, telegramID, username, "", false
	}

	if user == nil || user.AccountDetailsLink == "" {
		link = api.GenerateAccountLink(username, telegramID)
		if err := db.CreateVpnUser(ctx, username, telegramID, link, false); err != nil {
			log.Printf("failed to create vpn user: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Sorry, something went wrong while creating your account."))
			return ctx, telegramID, username, "", false
		}
	} else {
		link = user.AccountDetailsLink
	}
	return ctx, telegramID, username, link, true
}

