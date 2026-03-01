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
```

**How to get `CRYPTO_PAY_API_TOKEN`:**

- **Mainnet (real payments):** Open [@CryptoBot](https://t.me/CryptoBot) → **Crypto Pay** → **Create App** → copy the API token.
- **Testnet (testing only):** Open [@CryptoTestnetBot](https://t.me/CryptoTestnetBot) → **Crypto Pay** → **Create App** → copy the API token. Use that token as `CRYPTO_PAY_API_TOKEN` and set `CRYPTO_PAY_TESTNET=1`.

You can also make them permanent using `setx`, but for development the per-session variables are usually enough.

## 4. Install Go dependencies

From the project root:

```bash
go mod tidy
```

This downloads all required Go modules (Telegram API client, pgx, etc.).

## 5. Run the bot

From the project root:

```bash
go run .
```

On startup the app runs DB migrations from `./migrations` (using `DATABASE_URL`).

You should see a log line similar to:

```text
Authorized on account <bot_username>
```

The bot is now running and listening for messages.

## 6. Available commands

You can type commands or use the reply keyboard buttons after `/start`.

| Command / Button | Description |
|------------------|-------------|
| `/start` | Welcome message and reply keyboard: **Guide**, **Profile**, **Subscription**. Ensures the user exists in the DB and has an account link (creates one via Remnawave if needed). |
| `/instruction` (Guide) | Sends step-by-step instructions (editable in `main.go`). |
| `/account` (Profile) | Returns your VPN account details link. If you don’t have one yet, the bot creates a Remnawave user and stores the link. |
| `/substribe` (Subscription) | Opens subscription flow (see below). |

### Subscription flow

1. **Choose a plan:** 1 week test period, 1 month (249 ₽), or 3 months (699 ₽).
2. **1 week test** — Applied immediately (no payment). One-time; if already used, the bot says so.
3. **1 month / 3 months** — Choose payment method:
   - **Pay Telegram Stars** — Bot sends a Telegram Stars (XTR) invoice. After payment, the subscription is applied in Remnawave.
   - **Pay crypto** — Bot creates a Crypto Pay invoice and sends the payment link (requires `CRYPTO_PAY_API_TOKEN`).

## 7. Stopping the services

To stop the Postgres container:

```bash
docker compose down
```

To stop the bot, press `Ctrl + C` in the terminal where `go run .` is running.