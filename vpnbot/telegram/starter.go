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
			switch update.CallbackQuery.Data {
			case "plan_1week":
				// 1 week test: apply immediately, no payment step
				HandleSub1WeekTest(bot, update.CallbackQuery)
			case "plan_1month":
				HandlePlanChoice(bot, update.CallbackQuery, "1month")
			case "plan_3months":
				HandlePlanChoice(bot, update.CallbackQuery, "3months")
			case "pay_stars_1month":
				HandlePayStarsPlan1Month(bot, update.CallbackQuery)
			case "pay_stars_3months":
				HandlePayStarsPlan3Months(bot, update.CallbackQuery)
			case "pay_crypto_1month":
				HandlePayCryptoPlan1Month(bot, update.CallbackQuery)
			case "pay_crypto_3months":
				HandlePayCryptoPlan3Months(bot, update.CallbackQuery)
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
		case "Help":
			update.Message.Text = "/help"
		}

		if update.Message.IsCommand() || strings.HasPrefix(update.Message.Text, "/") {
			HandleCommand(bot, update.Message)
			continue
		}

		HandleMessage(bot, update.Message)
	}
}

