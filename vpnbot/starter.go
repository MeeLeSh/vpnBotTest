package vpnbot

import (
	"context"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Run() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	AppConfig = cfg

	if err := runMigrations(AppConfig.DatabaseURL); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	db, err = initDB(context.Background())
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer db.Close()

	if err := initRemnawaveClient(); err != nil {
		log.Fatalf("failed to init Remnawave client: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(AppConfig.TelegramBotToken)
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
			handlePreCheckoutQuery(bot, update.PreCheckoutQuery)
			continue
		}

		// Successful payment: deliver subscription
		if update.Message != nil && update.Message.SuccessfulPayment != nil {
			handleSuccessfulStarsPayment(bot, update.Message)
			continue
		}

		// Handle callback queries from inline buttons
		if update.CallbackQuery != nil {
			switch update.CallbackQuery.Data {
			case "plan_1week":
				// 1 week test: apply immediately, no payment step
				handleSub1WeekTest(bot, update.CallbackQuery)
			case "plan_1month":
				handlePlanChoice(bot, update.CallbackQuery, "1month")
			case "plan_3months":
				handlePlanChoice(bot, update.CallbackQuery, "3months")
			case "pay_stars_1month":
				handlePayStarsPlan1Month(bot, update.CallbackQuery)
			case "pay_stars_3months":
				handlePayStarsPlan3Months(bot, update.CallbackQuery)
			case "pay_crypto_1month":
				handlePayCryptoPlan1Month(bot, update.CallbackQuery)
			case "pay_crypto_3months":
				handlePayCryptoPlan3Months(bot, update.CallbackQuery)
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
			handleCommand(bot, update.Message)
			continue
		}

		handleMessage(bot, update.Message)
	}
}

