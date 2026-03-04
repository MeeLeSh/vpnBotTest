package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpnBot/vpnbot/api"
	"vpnBot/vpnbot/config"
	"vpnBot/vpnbot/db"
)

// newMainReplyKeyboard returns the main menu keyboard (Guide, Profile, Subscription, Help, Questions for admin).
func newMainReplyKeyboard(user *tgbotapi.User) tgbotapi.ReplyKeyboardMarkup {
	row1 := tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Guide"),
		tgbotapi.NewKeyboardButton("Profile"),
	)
	row2Buttons := []tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButton("Subscription"),
		tgbotapi.NewKeyboardButton("Help"),
	}
	if config.AppConfig != nil && config.AppConfig.AdminTelegramID != 0 && user != nil && int64(user.ID) == config.AppConfig.AdminTelegramID {
		row2Buttons = append(row2Buttons, tgbotapi.NewKeyboardButton("Questions"))
	}
	row2 := tgbotapi.NewKeyboardButtonRow(row2Buttons...)
	return tgbotapi.NewReplyKeyboard(row1, row2)
}

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
	case "cancel":
		handleCancelCommand(bot, msg)
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

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
	bot.Send(reply)
}

func handleCancelCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	if IsAwaitingHelpQuestion(chatID) {
		ClearAwaitingHelpQuestion(chatID)
		reply := tgbotapi.NewMessage(chatID, "Question cancelled. You can continue using the menu.")
		reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
		bot.Send(reply)
		return
	}
	if _, ok := GetAwaitingAnswer(chatID); ok {
		ClearAwaitingAnswer(chatID)
		bot.Send(tgbotapi.NewMessage(chatID, "Answer cancelled."))
		return
	}
	bot.Send(tgbotapi.NewMessage(chatID, "There is nothing to cancel."))
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
	keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Cancel")))
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Write your question to the chat")
	reply.ReplyMarkup = keyboard
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
	questions, err := db.GetUnansweredUserQuestions(ctx)
	if err != nil {
		log.Printf("failed to load user questions: %v", err)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Failed to load questions. Please try again later.")
		bot.Send(reply)
		return
	}

	if len(questions) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "There are no unanswered questions.")
		bot.Send(reply)
		return
	}

	for _, q := range questions {
		text := fmt.Sprintf("Question:\n%s\n\nTelegram ID: %d", q.Message, q.TelegramID)
		answerBtn := tgbotapi.NewInlineKeyboardButtonData("Answer", fmt.Sprintf("reply_to:%d", q.ID))
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(answerBtn))
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

// HandleState processes non-command messages: help question, admin answer, or default.
func HandleState(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	ctx := context.Background()

	// Admin sending answer to a question (after clicking Answer).
	if questionID, ok := GetAwaitingAnswer(msg.Chat.ID); ok && msg.Text != "" {
		ClearAwaitingAnswer(msg.Chat.ID)
		q, err := db.GetUserQuestionByID(ctx, questionID)
		if err != nil || q == nil {
			log.Printf("get question %d: %v", questionID, err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Failed to load question. Try again."))
			return
		}
		if err := db.UpdateUserQuestionAnswer(ctx, questionID, msg.Text); err != nil {
			log.Printf("update question answer: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Failed to save answer. Try again."))
			return
		}
		outToUser := fmt.Sprintf("Your question:\n%s\n\nAnswer:\n%s", q.Message, msg.Text)
		if _, err := bot.Send(tgbotapi.NewMessage(q.TelegramID, outToUser)); err != nil {
			log.Printf("send answer to user %d: %v", q.TelegramID, err)
		}
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Answer sent to the user and saved.")
		reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
		if _, err := bot.Send(reply); err != nil {
			log.Printf("failed to send answer confirmation to admin: %v", err)
		}
		return
	}

	// User sending their question (after Help).
	if IsAwaitingHelpQuestion(msg.Chat.ID) && msg.Text != "" {
		ClearAwaitingHelpQuestion(msg.Chat.ID)
		msgTime := time.Unix(int64(msg.Date), 0)
		if err := db.InsertUserQuestion(ctx, int64(msg.From.ID), msg.Text, msgTime); err != nil {
			log.Printf("failed to save user question: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Sorry, we couldn't save your question. Please try again later."))
		} else {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Your question has been saved. We'll get back to you soon.")
			reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
			bot.Send(reply)
		}
		return
	}

	HandleMessage(bot, msg)
}

// HandleStartReplyToUser is called when admin taps "Answer" under a question. Sets state and asks for answer text.
func HandleStartReplyToUser(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	if _, err := bot.Request(tgbotapi.NewCallback(callback.ID, "")); err != nil {
		log.Printf("answer callback: %v", err)
	}
	parts := strings.SplitN(callback.Data, ":", 2)
	if len(parts) != 2 {
		return
	}
	questionID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}
	SetAwaitingAnswer(callback.Message.Chat.ID, questionID)
	keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Cancel")))
	reply := tgbotapi.NewMessage(callback.Message.Chat.ID, "Write your answer and send it, or tap Cancel to cancel.")
	reply.ReplyMarkup = keyboard
	bot.Send(reply)
}

// HandleCancelReplyCallback handles inline Cancel during answer flow (if we add inline Cancel; currently Cancel is reply keyboard).
func HandleCancelReplyCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	if _, err := bot.Request(tgbotapi.NewCallback(callback.ID, "")); err != nil {
		log.Printf("answer callback: %v", err)
	}
	ClearAwaitingAnswer(callback.Message.Chat.ID)
	bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "Answer cancelled."))
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

