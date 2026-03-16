FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically linked Linux binary. Use a name that does not clash
# with the source directory `vpnbot`.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/app ./cmd/vpnbot

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /app/app ./app
# Copy database migrations so golang-migrate can load "file://migrations"
COPY migrations ./migrations

USER appuser

ENV TELEGRAM_BOT_TOKEN=""
ENV DATABASE_URL=""
ENV REMNAWAVE_PANEL_URL=""
ENV REMNAWAVE_API_TOKEN=""
ENV REMNAWAVE_SQUAD_ID=""
ENV CRYPTO_PAY_API_TOKEN=""
ENV CRYPTO_PAY_TESTNET=""
ENV TELEGRAM_ADMIN_ID=""
ENV GUIDE_TEXT=""

CMD ["./app"]

