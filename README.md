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
```

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

- `/start`  
Short description of the bot and a hint to use `/instruction`.
- `/instruction`  
Sends step-by-step instructions (editable in `main.go`).
- `/account`  
  - If the user **does not exist** in the `VpnUser` table:
    - Computes a username (Telegram `UserName` or `tg-<telegramId>` as fallback)
    - Generates a new account link using username
    - Calls Remnawave `Users().CreateUser` with that username, `ExpireAt = now()`, and `TelegramId`
    - Inserts a new row into `VpnUser`
    - Returns the generated link.
  - If the user **exists** in `VpnUser`:
    - Reads `accountDetailsLink` from the database
    - Returns that stored link.

## 7. Stopping the services

To stop the Postgres container:

```bash
docker compose down
```

To stop the bot, press `Ctrl + C` in the terminal where `go run .` is running.