package adapter

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StripeAdapter defines the Anti-Corruption Layer interface for Stripe payment operations.
// This abstraction decouples the domain from the external Stripe API.
type StripeAdapter interface {
	// CreatePaymentIntent creates a Stripe PaymentIntent with manual capture (authorize only).
	CreatePaymentIntent(ctx context.Context, amountCents int64, currency, customerEmail string) (paymentIntentID, clientSecret string, err error)

	// CapturePaymentIntent captures a previously authorized PaymentIntent.
	CapturePaymentIntent(ctx context.Context, paymentIntentID string) error

	// CancelPaymentIntent cancels an uncaptured PaymentIntent.
	CancelPaymentIntent(ctx context.Context, paymentIntentID string) error

	// CreateRefund refunds a captured PaymentIntent.
	CreateRefund(ctx context.Context, paymentIntentID string, amountCents int64) error
}

// MockStripeAdapter is a development/testing implementation of StripeAdapter.
// It simulates Stripe behavior without requiring a real Stripe account.
type MockStripeAdapter struct {
	logger *zap.Logger
}

// NewMockStripeAdapter creates a new mock Stripe adapter for development.
func NewMockStripeAdapter(logger *zap.Logger) *MockStripeAdapter {
	return &MockStripeAdapter{logger: logger}
}

// CreatePaymentIntent simulates creating a PaymentIntent and returns mock IDs.
func (m *MockStripeAdapter) CreatePaymentIntent(ctx context.Context, amountCents int64, currency, customerEmail string) (string, string, error) {
	paymentIntentID := fmt.Sprintf("pi_mock_%s", uuid.New().String()[:8])
	clientSecret := fmt.Sprintf("%s_secret_mock", paymentIntentID)

	m.logger.Info("[MOCK STRIPE] PaymentIntent created",
		zap.String("payment_intent_id", paymentIntentID),
		zap.Int64("amount_cents", amountCents),
		zap.String("currency", currency),
		zap.String("customer_email", customerEmail),
	)

	return paymentIntentID, clientSecret, nil
}

// CapturePaymentIntent simulates capturing a PaymentIntent.
func (m *MockStripeAdapter) CapturePaymentIntent(ctx context.Context, paymentIntentID string) error {
	m.logger.Info("[MOCK STRIPE] PaymentIntent captured",
		zap.String("payment_intent_id", paymentIntentID),
	)
	return nil
}

// CancelPaymentIntent simulates cancelling a PaymentIntent.
func (m *MockStripeAdapter) CancelPaymentIntent(ctx context.Context, paymentIntentID string) error {
	m.logger.Info("[MOCK STRIPE] PaymentIntent cancelled",
		zap.String("payment_intent_id", paymentIntentID),
	)
	return nil
}

// CreateRefund simulates refunding a PaymentIntent.
func (m *MockStripeAdapter) CreateRefund(ctx context.Context, paymentIntentID string, amountCents int64) error {
	m.logger.Info("[MOCK STRIPE] Refund created",
		zap.String("payment_intent_id", paymentIntentID),
		zap.Int64("amount_cents", amountCents),
	)
	return nil
}
