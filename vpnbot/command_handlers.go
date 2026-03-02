package vpnbot

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
	case "help":
		handleHelpCommand(bot, msg)
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

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Guide"),
			tgbotapi.NewKeyboardButton("Profile"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Subscription"),
			tgbotapi.NewKeyboardButton("Help"),
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
	_, _, _, link, ok := resolveUserAndLink(bot, msg)
	if !ok {
		return
	}

	text := "Your account details are available here:\n" + link
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.Send(reply)
}

func handleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "write your question to the chat")
	bot.Send(reply)
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

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, " ")
	bot.Send(reply)
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

	user, err := getVpnUserByUsername(ctx, username)
	if err != nil {
		log.Printf("failed to get vpn user: %v", err)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Sorry, something went wrong. Please try again later."))
		return ctx, telegramID, username, "", false
	}

	if user == nil || user.AccountDetailsLink == "" {
		link = generateAccountLink(username, telegramID)
		if err := createVpnUser(ctx, username, telegramID, link, false); err != nil {
			log.Printf("failed to create vpn user: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Sorry, something went wrong while creating your account."))
			return ctx, telegramID, username, "", false
		}
	} else {
		link = user.AccountDetailsLink
	}
	return ctx, telegramID, username, link, true
}


