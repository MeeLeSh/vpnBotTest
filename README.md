# vpnBot

Simple Telegram bot written in Go that manages VPN user accounts and stores account links in PostgreSQL (running in Docker).

## Prerequisites

- Go 1.26+
- Docker & Docker Compose
- Telegram bot token from BotFather

## 1. Clone / open the project

Go to the `vpnBot` project directory (for example using your file explorer, or with a `cd` command to the appropriate path).

## 2. Set environment variables

In a terminal where you will run the bot (PowerShell examples):

```powershell
$env:TELEGRAM_BOT_TOKEN   = "YOUR_TELEGRAM_TOKEN_HERE"
$env:DATABASE_URL         = "postgres://vpn_user:vpn_password@localhost:5433/vpn_db?sslmode=disable"
$env:REMNAWAVE_PANEL_URL  = "https://your-remnawave-panel.example.com"
$env:REMNAWAVE_API_TOKEN  = "YOUR_REMNAWAVE_JWT_TOKEN"
# Optional: internal squad UUID to assign to user when they subscribe (Remnawave panel)
$env:REMNAWAVE_SQUAD_ID   = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
# Optional: Crypto Pay (for "Pay crypto" subscription)
$env:CRYPTO_PAY_API_TOKEN = "YOUR_CRYPTO_PAY_APP_TOKEN"
$env:CRYPTO_PAY_TESTNET   = "1"                           # set to 1 for testnet
$env:TELEGRAM_ADMIN_ID    = "123456789"                   # optional: numeric Telegram user ID for admin-only Вопросы/Рассылка buttons
# Optional: text sent when user taps «Инструкция»
# Use \n for newlines; the bot converts them to real line breaks.
$env:GUIDE_TEXT           = "Вот что нужно сделать:\n\n1. Шаг первый.\n2. Шаг второй.\n3. Шаг третий."
# Optional: Telegram Stars prices (positive integers)
$env:STARS_1_MONTH        = "1"
$env:STARS_3_MONTHS       = "3"
```

**How to get `CRYPTO_PAY_API_TOKEN`:**

- **Mainnet (real payments):** Open `@CryptoBot` in Telegram → **Crypto Pay** → **Create App** → copy the API token.
- **Testnet (testing only):** Open `@CryptoTestnetBot` → **Crypto Pay** → **Create App** → copy the API token. Use that token as `CRYPTO_PAY_API_TOKEN` and set `CRYPTO_PAY_TESTNET=1`.

You can also make them permanent using `setx`, but for development the per-session variables are usually enough.

## 3. Install Go dependencies

From the project root:

```bash
go mod tidy
```

This downloads all required Go modules (Telegram API client, pgx, etc.).

## 4. Run the bot

You can run the bot either directly with Go or via Docker Compose.

### Option A: Run with Go (local)

First, start PostgreSQL (for example, using Docker Compose):

```bash
docker compose up -d postgres
```

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

### Option B: Run bot + Postgres with Docker Compose

From the project root (with required environment variables set in the shell or a `.env` file):

```bash
docker compose up -d
```

This will:

- start PostgreSQL (`postgres` service),
- build the Go bot image from the `Dockerfile`,
- start the `vpnbot` service after Postgres.

### (Optional) Build a binary

From the project root:

```bash
go build -o vpnbot.exe ./cmd/vpnbot
.\vpnbot.exe
```

## 5. Available commands

You can type commands or use the reply keyboard buttons after `/start`.

| Command / Button | Description |
|------------------|-------------|
| `/start` | Welcome message and reply keyboard: **Инструкция**, **Профиль**, **Подписка**, **Помощь** (and **Вопросы** / **Рассылка** for admin). Ensures the user exists in the DB and has an account link (creates one via Remnawave if needed). |
| `/instruction` (Инструкция) | Sends step-by-step instructions. Text is set via **`GUIDE_TEXT`** (env); if unset, a default Russian placeholder is used. |
| `/account` (Профиль) | Returns your VPN account details link. If you don’t have one yet, the bot creates a Remnawave user and stores the link. |
| `/substribe` (Подписка) | Opens subscription flow (see below). |
| `/help` (Помощь) | Asks the user to write a question; shows an **Отмена** button. The next message is saved as a question in the DB, or **Отмена** aborts. Main keyboard is restored after sending. |
| `/cancel` (Отмена) | Cancels pending "write question", "write answer", or "broadcast" state; restores main menu or confirms cancellation to admin. |
| `/questions` (Вопросы, admin only) | Shown when `TELEGRAM_ADMIN_ID` is set. Lists **unanswered** questions only; each has an **Ответить** inline button. |
| `/broadcast` (Рассылка, admin only) | Shown when `TELEGRAM_ADMIN_ID` is set. Asks admin to type a message and sends it to all users stored in `VpnUser`; **Отмена** aborts. |

### Subscription flow

1. **Choose a plan:** 1 week test period, 1 month (249 ₽), or 3 months (699 ₽).
2. **1 week test** — Applied immediately (no payment). One-time per user; the bot warns if already used.
3. **1 month / 3 months** — Choose payment method:
   - Telegram Stars — Bot sends a Telegram Stars (XTR) invoice. After payment, the subscription is applied in Remnawave.
   - Crypto Pay — Bot creates a Crypto Pay invoice and sends the payment link (requires `CRYPTO_PAY_API_TOKEN`).

On subscribe (test or paid), the bot in Remnawave: extends the user's expiry, sets **HWID device limit to 3**, and if **`REMNAWAVE_SQUAD_ID`** is set it replaces the user's active internal squads with **exactly that one squad**. If `REMNAWAVE_SQUAD_ID` is unset, squads are left unchanged.

### Help & user questions

- **User:** Press **Помощь** → bot asks the user to write a question and shows **Отмена**. The next message is saved to `UserQuestion`, or **Отмена** cancels.
- **Admin:** Press **Вопросы** → bot sends only **unanswered** questions. Each block has question text, Telegram ID, and an **Ответить** button.
- **Admin answers:** Tap **Ответить** → bot asks to type the answer and shows **Отмена**. Admin sends a message → it is **stored in the DB** (`UserQuestion.answer`) and sent back to the user together with their original question; the question disappears from the unanswered list. Or tap **Отмена** to abort.

## 6. Stopping the services

To stop the Postgres container:

```bash
docker compose down
```

To stop the bot, press `Ctrl + C` in the terminal where `go run ./cmd/vpnbot` is running.