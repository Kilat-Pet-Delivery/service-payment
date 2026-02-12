package subscription

import (
	"context"

	"github.com/google/uuid"
)

// SubscriptionRepository defines persistence operations for subscriptions.
type SubscriptionRepository interface {
	Save(ctx context.Context, s *Subscription) error
	Update(ctx context.Context, s *Subscription) error
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Subscription, error)
}
