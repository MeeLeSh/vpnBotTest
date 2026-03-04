package telegram

import (
	"context"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"vpnBot/vpnbot/api"
	"vpnBot/vpnbot/config"
	"vpnBot/vpnbot/db"
)

func Run() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	config.AppConfig = cfg

	if err := db.RunMigrations(config.AppConfig.DatabaseURL); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	if _, err := db.Init(context.Background(), config.AppConfig.DatabaseURL); err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer db.Close()

	if err := api.InitRemnawaveClient(); err != nil {
		log.Fatalf("failed to init Remnawave client: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(config.AppConfig.TelegramBotToken)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	go RunScheduler(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// Pre-checkout: must answer within 10 seconds or payment is cancelled
		if update.PreCheckoutQuery != nil {
			HandlePreCheckoutQuery(bot, update.PreCheckoutQuery)
			continue
		}

		// Successful payment: deliver subscription
		if update.Message != nil && update.Message.SuccessfulPayment != nil {
			HandleSuccessfulStarsPayment(bot, update.Message)
			continue
		}

		// Handle callback queries from inline buttons
		if update.CallbackQuery != nil {
			data := update.CallbackQuery.Data
			switch {
			case data == "plan_1week":
				HandleSub1WeekTest(bot, update.CallbackQuery)
			case data == "plan_1month":
				HandlePlanChoice(bot, update.CallbackQuery, "1month")
			case data == "plan_3months":
				HandlePlanChoice(bot, update.CallbackQuery, "3months")
			case data == "pay_stars_1month":
				HandlePayStarsPlan1Month(bot, update.CallbackQuery)
			case data == "pay_stars_3months":
				HandlePayStarsPlan3Months(bot, update.CallbackQuery)
			case data == "pay_crypto_1month":
				HandlePayCryptoPlan1Month(bot, update.CallbackQuery)
			case data == "pay_crypto_3months":
				HandlePayCryptoPlan3Months(bot, update.CallbackQuery)
			case strings.HasPrefix(data, "reply_to:"):
				HandleStartReplyToUser(bot, update.CallbackQuery)
			}
			continue
		}

		if update.Message == nil {
			continue
		}

		// Map button labels to commands
		switch update.Message.Text {
		case "Guide":
			update.Message.Text = "/instruction"
		case "Profile":
			update.Message.Text = "/account"
		case "Subscription":
			update.Message.Text = "/substribe"
		case "Help":
			update.Message.Text = "/help"
		case "Questions":
			update.Message.Text = "/questions"
		case "Cancel":
			update.Message.Text = "/cancel"
		}

		if update.Message.IsCommand() || strings.HasPrefix(update.Message.Text, "/") {
			HandleCommand(bot, update.Message)
			continue
		}

		// Delegate non-command message handling (including help state) to command handlers.
		HandleState(bot, update.Message)
	}
}

