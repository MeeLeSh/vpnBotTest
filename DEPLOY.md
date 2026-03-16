## Deployment guide

This file summarizes how to deploy and run `vpnBot` in different environments.

### 1. Requirements

- Go 1.26+ (if you build/run locally)
- Docker and Docker Compose (for containerized deployment)
- Telegram bot token
- Remnawave panel URL and API token (if you use Remnawave integration)
- Optional: Crypto Pay API token (for crypto payments)

### 2. Environment variables

The bot is configured entirely via environment variables:

- `TELEGRAM_BOT_TOKEN` – Telegram bot token from BotFather (required)
- `DATABASE_URL` – PostgreSQL connection string
- `REMNAWAVE_PANEL_URL` – Remnawave panel base URL
- `REMNAWAVE_API_TOKEN` – Remnawave JWT token
- `REMNAWAVE_SQUAD_ID` – internal squad UUID to assign on subscribe (optional)
- `CRYPTO_PAY_API_TOKEN` – Crypto Pay API token (optional)
- `CRYPTO_PAY_TESTNET` – `1` for Crypto Pay testnet, empty/`0` for mainnet
- `TELEGRAM_ADMIN_ID` – Telegram user ID of admin (for Вопросы/Рассылка)
- `GUIDE_TEXT` – text for the **Инструкция** button (supports `\n` as line breaks)

For Docker Compose, you can put these into a `.env` file in the project root; Compose will pick them up automatically.

### 3. Deploy with Docker Compose (recommended)

1. **Build and start services** from the project root:

   ```bash
   docker compose up -d
   ```

   This will:

   - start PostgreSQL (`postgres` service),
   - build the Go bot image using `Dockerfile`,
   - start the `vpnbot` service **after** Postgres passes its healthcheck.

2. **View logs**:

   ```bash
   docker compose logs -f vpnbot
   ```

   When you see:

   ```text
   Authorized on account <bot_username>
   ```

   the bot is up and connected to Telegram.

3. **Stop services**:

   ```bash
   docker compose down
   ```

### 4. Deploy as a binary (without Docker)

1. Build the binary:

   ```bash
   go build -o vpnbot.exe ./cmd/vpnbot
   ```

2. Make sure PostgreSQL is running and `DATABASE_URL` points to it.

3. Set all required environment variables (see section 2 above).

4. Run the bot:

   ```bash
   ./vpnbot.exe
   ```

The bot will run until you stop the process (for example with `Ctrl + C` or a service manager on the server).

