// Package api: Crypto Pay API client (https://help.send.tg/en/articles/10279948-crypto-pay-api).
// API base: https://pay.crypt.bot (mainnet), https://testnet-pay.crypt.bot (testnet).

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"vpnBot/vpnbot/config"
)

// Invoice is the Crypto Pay API Invoice object returned by createInvoice.
// See https://help.send.tg/en/articles/10279948-crypto-pay-api
type Invoice struct {
	InvoiceID         int64  `json:"invoice_id"`
	Hash              string `json:"hash"`
	CurrencyType      string `json:"currency_type"`
	Asset             string `json:"asset,omitempty"`
	Fiat              string `json:"fiat,omitempty"`
	Amount            string `json:"amount"`
	BotInvoiceURL     string `json:"bot_invoice_url"`
	MiniAppInvoiceURL string `json:"mini_app_invoice_url,omitempty"`
	WebAppInvoiceURL  string `json:"web_app_invoice_url,omitempty"`
	Description       string `json:"description,omitempty"`
	Status            string `json:"status"`
	Payload           string `json:"payload,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	ExpirationDate    string `json:"expiration_date,omitempty"`
}

// CreateInvoiceOpts are options for creating a Crypto Pay invoice.
type CreateInvoiceOpts struct {
	CurrencyType  string `json:"currency_type,omitempty"` // "crypto" or "fiat", default "crypto"
	Asset         string `json:"asset,omitempty"`         // required if currency_type is "crypto": USDT, TON, BTC, etc.
	Fiat          string `json:"fiat,omitempty"`          // required if currency_type is "fiat": RUB, USD, EUR, etc.
	Amount        string `json:"amount"`                  // e.g. "125.50"
	Description   string `json:"description,omitempty"`
	HiddenMessage string `json:"hidden_message"`
	Payload       string `json:"payload,omitempty"`
	ExpiresIn     int    `json:"expires_in,omitempty"` // seconds 1–2678400
}

type createInvoiceResponse struct {
	OK     bool            `json:"ok"`
	Result Invoice         `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"` // API may return string or object
}

type getInvoicesResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"` // API may return array or object
	Error  json.RawMessage `json:"error,omitempty"`
}

var cryptoPayHTTPClient = &http.Client{Timeout: 15 * time.Second}

// CreateInvoice creates a new invoice via Crypto Pay API and returns the Invoice.
// Uses AppConfig.CryptoPayToken and AppConfig.CryptoPayTestnet (env CRYPTO_PAY_API_TOKEN, CRYPTO_PAY_TESTNET).
// Returns (nil, nil) if token is not set or config not loaded.
func CreateInvoice(ctx context.Context, opts CreateInvoiceOpts) (*Invoice, error) {
	if config.AppConfig == nil || config.AppConfig.CryptoPayToken == "" {
		return nil, nil
	}

	base := config.AppConfig.CryptoPayAPIBaseURL
	if config.AppConfig.CryptoPayTestnet == "1" {
		base = config.AppConfig.CryptoPayAPITestnetURL
	}

	body, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("cryptopay: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/createInvoice", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cryptopay: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Crypto-Pay-API-Token", config.AppConfig.CryptoPayToken)

	resp, err := cryptoPayHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cryptopay: do request: %w", err)
	}
	defer resp.Body.Close()

	var out createInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("cryptopay: decode response: %w", err)
	}

	if !out.OK {
		errMsg := string(out.Error)
		if errMsg == "" {
			errMsg = "unknown"
		}
		return nil, fmt.Errorf("cryptopay: api error: %s", errMsg)
	}

	return &out.Result, nil
}

// GetInvoices returns invoices created by your app for the given invoice IDs (see https://help.send.tg/en/articles/10279948-crypto-pay-api).
// Returns (nil, nil) if token is not set, config not loaded, or invoiceIDs is empty.
func GetInvoices(ctx context.Context, invoiceIDs []int64) ([]Invoice, error) {
	if config.AppConfig == nil || config.AppConfig.CryptoPayToken == "" {
		return nil, nil
	}
	if len(invoiceIDs) == 0 {
		return nil, nil
	}

	base := config.AppConfig.CryptoPayAPIBaseURL
	if config.AppConfig.CryptoPayTestnet == "1" {
		base = config.AppConfig.CryptoPayAPITestnetURL
	}

	// build invoice_ids param as comma-separated list
	idsStr := make([]string, len(invoiceIDs))
	for i, id := range invoiceIDs {
		idsStr[i] = strconv.FormatInt(id, 10)
	}
	values := url.Values{}
	values.Set("invoice_ids", strings.Join(idsStr, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/getInvoices?"+values.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("cryptopay: new request: %w", err)
	}
	req.Header.Set("Crypto-Pay-API-Token", config.AppConfig.CryptoPayToken)

	resp, err := cryptoPayHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cryptopay: do request: %w", err)
	}
	defer resp.Body.Close()

	var out getInvoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("cryptopay: decode getInvoices response: %w", err)
	}

	if !out.OK {
		errMsg := string(out.Error)
		if errMsg == "" {
			errMsg = "unknown"
		}
		return nil, fmt.Errorf("cryptopay: api error: %s", errMsg)
	}

	var invoices []Invoice
	if len(out.Result) == 0 {
		return invoices, nil
	}

	// result can be either a JSON array of invoices or an object with "items":[...]
	// try array first
	if err := json.Unmarshal(out.Result, &invoices); err == nil {
		return invoices, nil
	}

	// then try {"items":[...]}
	var withItems struct {
		Items []Invoice `json:"items"`
	}
	if err := json.Unmarshal(out.Result, &withItems); err != nil {
		log.Printf("cryptopay: getInvoices unmarshal result failed, raw=%s, err=%v", string(out.Result), err)
		return nil, nil
	}

	return withItems.Items, nil
}

