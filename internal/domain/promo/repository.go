package promo

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PromoRepository defines persistence operations for promo codes.
type PromoRepository interface {
	Save(ctx context.Context, p *PromoCode) error
	Update(ctx context.Context, p *PromoCode) error
	FindByCode(ctx context.Context, code string) (*PromoCode, error)
	FindByID(ctx context.Context, id uuid.UUID) (*PromoCode, error)
	FindActive(ctx context.Context) ([]*PromoCode, error)
	SaveUsage(ctx context.Context, usage *PromoUsage) error
	HasUserUsedPromo(ctx context.Context, promoID, userID uuid.UUID) (bool, error)
}

// PromoUsage tracks each individual promo code usage.
type PromoUsage struct {
	ID            uuid.UUID
	PromoID       uuid.UUID
	UserID        uuid.UUID
	BookingID     uuid.UUID
	DiscountCents int64
	UsedAt        time.Time
}
