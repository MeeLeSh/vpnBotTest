# Telegram Stars (payments) in vpnBot

## Overview

[Telegram Stars](https://core.telegram.org/bots/payments-stars) is the only allowed way to sell **digital goods** (e.g. VPN subscription) inside Telegram. Users pay in Stars (currency code **XTR**); you get net proceeds later.

## Flow

1. **Send invoice** — `sendInvoice` with `currency: "XTR"`, no `provider_token` (empty string for digital goods).
2. **Pre-checkout** — User taps Pay → Telegram sends `pre_checkout_query`. Your bot must call `answerPreCheckoutQuery` within **10 seconds** (ok or error message).
3. **Success** — Telegram sends a message with `successful_payment`. Only then deliver the good (e.g. extend subscription).

## In this bot

- **1 week test** — remains free (no invoice).
- **1 month / 3 months** — send a Stars invoice; on `successful_payment` call `Subscribe(...)` and send confirmation.

## Implementation details (go-telegram-bot-api)

- **NewInvoice**: `providerToken` = `""`, `currency` = `"XTR"`, `prices` = `[]LabeledPrice{{Label: "1 month", Amount: N}}` (Amount = Stars, integer).
- **Pre-checkout**: handle `update.PreCheckoutQuery`, reply with `PreCheckoutConfig{PreCheckoutQueryID: query.ID, OK: true}` (or `OK: false` and `ErrorMessage`).
- **Delivery**: handle `update.Message != nil && update.Message.SuccessfulPayment != nil`, read `Payload` (e.g. `plan:1month`), call `Subscribe`, then send a thank-you message.

## Testing

Use Telegram’s [test environment](https://core.telegram.org/bots/features#testing-your-bot) for Stars; production uses real money.

## Live checklist (Telegram)

- Support: bot must handle `/support` or similar for payment issues.
- Terms: provide `/terms` or link to Terms and Conditions; users should accept before paying.
- Refunds: use [refundStarPayment](https://core.telegram.org/bots/api#refundstarpayment) if needed; store `telegram_payment_charge_id` from `SuccessfulPayment`.
