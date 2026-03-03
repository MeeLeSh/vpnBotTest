# vpnBot

Simple Telegram bot written in Go that manages VPN user accounts and stores account links in PostgreSQL (running in Docker).

## Prerequisites

- Go 1.26+
- Docker & Docker Compose
- Telegram bot token from BotFather

## 1. Clone / open the project

Make sure you are in the project folder:

```bash
cd m:\programming\projects\customers\vpnBot
```

## 2. Start PostgreSQL in Docker

This starts Postgres.
Database schema is managed by Go migrations in the `migrations/` folder and is applied automatically on app startup.

```bash
docker compose up -d
```

You only need to recreate containers if you change the Docker setup or wipe volumes.

## 3. Set environment variables

In a terminal where you will run the bot (PowerShell examples):

```powershell
$env:TELEGRAM_BOT_TOKEN   = "YOUR_TELEGRAM_TOKEN_HERE"
$env:DATABASE_URL         = "postgres://vpn_user:vpn_password@localhost:5433/vpn_db?sslmode=disable"
$env:REMNAWAVE_PANEL_URL  = "https://your-remnawave-panel.example.com"
$env:REMNAWAVE_API_TOKEN  = "YOUR_REMNAWAVE_JWT_TOKEN"
# Optional: Crypto Pay (for "Pay crypto" subscription)
$env:CRYPTO_PAY_API_TOKEN = "YOUR_CRYPTO_PAY_APP_TOKEN"
$env:CRYPTO_PAY_TESTNET   = "1"                           # set to 1 for testnet
$env:TELEGRAM_ADMIN_ID    = "123456789"                   # optional: numeric Telegram user ID for admin-only Questions button
```

**How to get `CRYPTO_PAY_API_TOKEN`:**

- **Mainnet (real payments):** Open `@CryptoBot` in Telegram → **Crypto Pay** → **Create App** → copy the API token.
- **Testnet (testing only):** Open `@CryptoTestnetBot` → **Crypto Pay** → **Create App** → copy the API token. Use that token as `CRYPTO_PAY_API_TOKEN` and set `CRYPTO_PAY_TESTNET=1`.

You can also make them permanent using `setx`, but for development the per-session variables are usually enough.

## 4. Install Go dependencies

From the project root:

```bash
go mod tidy
```

This downloads all required Go modules (Telegram API client, pgx, etc.).

## 5. Run the bot

The entrypoint is in `cmd/vpnbot/main.go`. From the project root:

```bash
go run ./cmd/vpnbot
```

On startup the app:

- loads config from environment (`vpnbot/config`),
- runs DB migrations from `./migrations` (using `DATABASE_URL`),
- opens a PostgreSQL connection pool (`vpnbot/db`),
- initializes Remnawave and Crypto Pay API clients (`vpnbot/api`),
- starts the Telegram update loop (`vpnbot/telegram`).

You should see a log line similar to:

```text
Authorized on account <bot_username>
```

The bot is now running and listening for messages.

### (Optional) Build a binary

From the project root:

```bash
go build -o vpnbot.exe ./cmd/vpnbot
.\vpnbot.exe
```

## 6. Available commands

You can type commands or use the reply keyboard buttons after `/start`.

| Command / Button | Description |
|------------------|-------------|
| `/start` | Welcome message and reply keyboard: **Guide**, **Profile**, **Subscription**. Ensures the user exists in the DB and has an account link (creates one via Remnawave if needed). |
| `/instruction` (Guide) | Sends step-by-step instructions (text is defined in `vpnbot/telegram/command_handlers.go`, in `handleInstructionCommand`). |
| `/account` (Profile) | Returns your VPN account details link. If you don’t have one yet, the bot creates a Remnawave user and stores the link. |
| `/substribe` (Subscription) | Opens subscription flow (see below). |
| `/help` (Help) | Asks the user to write a question; the next non-command message is stored in the `UserQuestion` table with Telegram ID and timestamp. |
| `/questions` (Questions, admin only) | Visible as a button only for the admin (when `TELEGRAM_ADMIN_ID` is set). Sends the admin a list of all stored questions and Telegram IDs, each with an **Answer** button that opens a chat with that user. |

### Subscription flow

1. **Choose a plan:** 1 week test period, 1 month (249 ₽), or 3 months (699 ₽).
2. **1 week test** — Applied immediately (no payment). One-time; if already used, the bot says so.
3. **1 month / 3 months** — Choose payment method:
   - **Pay Telegram Stars** — Bot sends a Telegram Stars (XTR) invoice. After payment, the subscription is applied in Remnawave.
   - **Pay crypto** — Bot creates a Crypto Pay invoice and sends the payment link (requires `CRYPTO_PAY_API_TOKEN`).

### Help & user questions

- When a user presses **Help** (or sends `/help`), the bot replies with a prompt to write a question.  
- The next plain text message from that user is saved into the `UserQuestion` table together with their Telegram ID and the message date.  
- When the admin presses **Questions** (or sends `/questions`), the bot sends blocks like:

  - question text  
  - Telegram ID  
  - an **Answer** inline button that opens the Telegram chat with that user.

## 7. Stopping the services

To stop the Postgres container:

```bash
docker compose down
```

To stop the bot, press `Ctrl + C` in the terminal where `go run ./cmd/vpnbot` is running.