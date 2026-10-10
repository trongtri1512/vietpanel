// Package billing provides payment gateway integrations for plan purchases.
// Supports SePay (VietQR) and Stripe as initial providers.
package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

type Provider string

const (
	ProviderSePay  Provider = "sepay"
	ProviderStripe Provider = "stripe"
)

type Plan struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    int64  `json:"price"` // VND for SePay, cents for Stripe
	Currency string `json:"currency"`
	Interval string `json:"interval"` // month, year
}

type Config struct {
	Provider     Provider
	SePayToken   string
	StripeKey    string
	WebhookSecret string
}

var cfg Config

func Init() {
	cfg = Config{
		Provider:     Provider(os.Getenv("BILLING_PROVIDER")),
		SePayToken:   os.Getenv("SEPAY_TOKEN"),
		StripeKey:    os.Getenv("STRIPE_KEY"),
		WebhookSecret: os.Getenv("BILLING_WEBHOOK_SECRET"),
	}
}

func VerifyWebhook(body []byte, signature string) bool {
	if cfg.WebhookSecret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}

func GetCheckoutURL(plan Plan, userID int64) (string, error) {
	switch cfg.Provider {
	case ProviderSePay:
		return fmt.Sprintf("https://sepay.vn/pay?amount=%d&info=Plan:%s:User:%d", plan.Price, plan.ID, userID), nil
	case ProviderStripe:
		return "", fmt.Errorf("stripe checkout not yet implemented")
	default:
		return "", fmt.Errorf("no billing provider configured")
	}
}

func AvailablePlans() []Plan {
	return []Plan{
		{ID: "starter", Name: "Starter", Price: 50000, Currency: "VND", Interval: "month"},
		{ID: "business", Name: "Business", Price: 200000, Currency: "VND", Interval: "month"},
		{ID: "enterprise", Name: "Enterprise", Price: 1000000, Currency: "VND", Interval: "month"},
	}
}