package payment

import (
	"context"

	"github.com/google/uuid"
)

// PaymentRepository defines the persistence contract for Payment aggregates.
type PaymentRepository interface {
	// FindByID retrieves a payment by its unique ID.
	FindByID(ctx context.Context, id uuid.UUID) (*Payment, error)

	// FindByBookingID retrieves a payment by the associated booking ID.
	FindByBookingID(ctx context.Context, bookingID uuid.UUID) (*Payment, error)

	// ListAll retrieves all payments with pagination (admin).
	ListAll(ctx context.Context, page, limit int) ([]*Payment, int64, error)

	// GetRevenueStats returns payment statistics (admin).
	GetRevenueStats(ctx context.Context) (totalRevenueCents int64, countByStatus map[string]int64, err error)

	// Save persists a new payment aggregate.
	Save(ctx context.Context, payment *Payment) error

	// Update persists changes to an existing payment aggregate with optimistic locking.
	Update(ctx context.Context, payment *Payment) error
}
