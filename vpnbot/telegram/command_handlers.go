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

// newMainReplyKeyboard returns the main menu keyboard (Инструкция, Профиль, Подписка, Помощь, Вопросы и Рассылка для админа).
func newMainReplyKeyboard(user *tgbotapi.User) tgbotapi.ReplyKeyboardMarkup {
	row1 := tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Инструкция"),
		tgbotapi.NewKeyboardButton("Профиль"),
	)
	row2 := tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Подписка"),
		tgbotapi.NewKeyboardButton("Помощь"),
	)
	if config.AppConfig != nil && config.AppConfig.AdminTelegramID != 0 && user != nil && int64(user.ID) == config.AppConfig.AdminTelegramID {
		row3 := tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Вопросы"),
			tgbotapi.NewKeyboardButton("Рассылка"),
		)
		return tgbotapi.NewReplyKeyboard(row1, row2, row3)
	}
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
	case "broadcast":
		handleBroadcastCommand(bot, msg)
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

	text := "Добро пожаловать! Этот бот поможет вам управлять VPN и вашим аккаунтом.\n" +
		"Для подробной пошаговой инструкции используйте кнопки ниже или команды."

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
	bot.Send(reply)
}

func handleCancelCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	if IsAwaitingHelpQuestion(chatID) {
		ClearAwaitingHelpQuestion(chatID)
		reply := tgbotapi.NewMessage(chatID, "Вопрос отменён. Вы можете продолжать пользоваться меню.")
		reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
		bot.Send(reply)
		return
	}
	if _, ok := GetAwaitingAnswer(chatID); ok {
		ClearAwaitingAnswer(chatID)
		reply := tgbotapi.NewMessage(chatID, "Ответ отменён.")
		reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
		bot.Send(reply)
		return
	}
	if IsAwaitingBroadcast(chatID) {
		ClearAwaitingBroadcast(chatID)
		reply := tgbotapi.NewMessage(chatID, "Рассылка отменена.")
		reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
		bot.Send(reply)
		return
	}
	bot.Send(tgbotapi.NewMessage(chatID, "Отменять сейчас нечего."))
}

func handleInstructionCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if config.AppConfig != nil && config.AppConfig.GuideText != "" {
		text = config.AppConfig.GuideText
		// Treat literal \n in env as newline (shells often don't expand escape sequences).
		text = strings.ReplaceAll(text, "\\n", "\n")
	}
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.Send(reply)
}

func handleAccountCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	_, _, _, link, ok := resolveUserAndLink(bot, msg)
	if !ok {
		return
	}

	text := "Ссылка на ваш VPN‑аккаунт:\n" + link
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	bot.Send(reply)
}

func handleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	SetAwaitingHelpQuestion(msg.Chat.ID)
	keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Отмена")))
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Напишите свой вопрос в этот чат.")
	reply.ReplyMarkup = keyboard
	bot.Send(reply)
}

func handleBroadcastCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if config.AppConfig == nil || config.AppConfig.AdminTelegramID == 0 || int64(msg.From.ID) != config.AppConfig.AdminTelegramID {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Эта команда доступна только администратору бота.")
		bot.Send(reply)
		return
	}
	SetAwaitingBroadcast(msg.Chat.ID)
	keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Отмена")))
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Введите сообщение для рассылки всем пользователям или нажмите «Отмена», чтобы прервать.")
	reply.ReplyMarkup = keyboard
	bot.Send(reply)
}

func handleQuestionsCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Only bot admin is allowed to see questions.
	if config.AppConfig == nil || config.AppConfig.AdminTelegramID == 0 || int64(msg.From.ID) != config.AppConfig.AdminTelegramID {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Эта команда доступна только администратору бота.")
		bot.Send(reply)
		return
	}

	ctx := context.Background()
	questions, err := db.GetUnansweredUserQuestions(ctx)
	if err != nil {
		log.Printf("failed to load user questions: %v", err)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Не удалось загрузить вопросы. Попробуйте позже.")
		bot.Send(reply)
		return
	}

	if len(questions) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Нет неотвеченных вопросов.")
		bot.Send(reply)
		return
	}

	for _, q := range questions {
		text := fmt.Sprintf("Вопрос:\n%s\n\nTelegram ID: %d", q.Message, q.TelegramID)
		answerBtn := tgbotapi.NewInlineKeyboardButtonData("Ответить", fmt.Sprintf("reply_to:%d", q.ID))
		markup := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(answerBtn))
		out := tgbotapi.NewMessage(msg.Chat.ID, text)
		out.ReplyMarkup = markup
		bot.Send(out)
	}
}

func handleSubstribeCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	text := "Выберите тариф подписки:"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Тест на 1 неделю", "plan_1week"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 месяц (249 ₽)", "plan_1month"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3 месяца (699 ₽)", "plan_3months"),
		),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = keyboard

	bot.Send(reply)
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Неизвестная команда. Попробуйте /start.")
	bot.Send(reply)
}

func HandleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, " ")
	bot.Send(reply)
}

// HandleState processes non-command messages: help question, admin answer, broadcast, or default.
func HandleState(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	ctx := context.Background()

	// Admin sending broadcast message (after tapping Send to all).
	if IsAwaitingBroadcast(msg.Chat.ID) && msg.Text != "" {
		ClearAwaitingBroadcast(msg.Chat.ID)
		ids, err := db.GetAllVpnUserTelegramIDs(ctx)
		if err != nil {
			log.Printf("failed to get user IDs for broadcast: %v", err)
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Не удалось получить список пользователей для рассылки. Попробуйте позже.")
			reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
			bot.Send(reply)
			return
		}
		var sent, failed int
		for _, id := range ids {
			if _, err := bot.Send(tgbotapi.NewMessage(id, msg.Text)); err != nil {
				log.Printf("broadcast to %d: %v", id, err)
				failed++
			} else {
				sent++
			}
		}
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Сообщение отправлено %d пользователям. Ошибок: %d.", sent, failed))
		reply.ReplyMarkup = newMainReplyKeyboard(msg.From)
		bot.Send(reply)
		return
	}

	// Admin sending answer to a question (after clicking Answer).
	if questionID, ok := GetAwaitingAnswer(msg.Chat.ID); ok && msg.Text != "" {
		ClearAwaitingAnswer(msg.Chat.ID)
		q, err := db.GetUserQuestionByID(ctx, questionID)
		if err != nil || q == nil {
			log.Printf("get question %d: %v", questionID, err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Не удалось загрузить вопрос. Попробуйте ещё раз."))
			return
		}
		if err := db.UpdateUserQuestionAnswer(ctx, questionID, msg.Text); err != nil {
			log.Printf("update question answer: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Не удалось сохранить ответ. Попробуйте ещё раз."))
			return
		}
		outToUser := fmt.Sprintf("Ваш вопрос:\n%s\n\nОтвет:\n%s", q.Message, msg.Text)
		if _, err := bot.Send(tgbotapi.NewMessage(q.TelegramID, outToUser)); err != nil {
			log.Printf("send answer to user %d: %v", q.TelegramID, err)
		}
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Ответ отправлен пользователю и сохранён.")
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
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Не удалось сохранить ваш вопрос. Пожалуйста, попробуйте позже."))
		} else {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Ваш вопрос сохранён. Мы ответим вам в ближайшее время.")
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
	keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Отмена")))
	reply := tgbotapi.NewMessage(callback.Message.Chat.ID, "Напишите ответ и отправьте его, или нажмите «Отмена», чтобы прервать.")
	reply.ReplyMarkup = keyboard
	bot.Send(reply)
}

// HandleCancelReplyCallback handles inline Cancel during answer flow (if we add inline Cancel; currently Cancel is reply keyboard).
func HandleCancelReplyCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	if _, err := bot.Request(tgbotapi.NewCallback(callback.ID, "")); err != nil {
		log.Printf("answer callback: %v", err)
	}
	ClearAwaitingAnswer(callback.Message.Chat.ID)
	bot.Send(tgbotapi.NewMessage(callback.Message.Chat.ID, "Ответ отменён."))
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
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Произошла ошибка. Пожалуйста, попробуйте позже."))
		return ctx, telegramID, username, "", false
	}

	if user == nil || user.AccountDetailsLink == "" {
		link = api.GenerateAccountLink(username, telegramID)
		if err := db.CreateVpnUser(ctx, username, telegramID, link, false); err != nil {
			log.Printf("failed to create vpn user: %v", err)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Произошла ошибка при создании аккаунта. Пожалуйста, попробуйте позже."))
			return ctx, telegramID, username, "", false
		}
	} else {
		link = user.AccountDetailsLink
	}
	return ctx, telegramID, username, link, true
}
