package application

import (
	"context"
	"fmt"
	"time"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	"github.com/Kilat-Pet-Delivery/lib-common/kafka"
	"github.com/Kilat-Pet-Delivery/lib-proto/events"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EventPublisher interface {
	PublishEvent(ctx context.Context, topic string, event kafka.CloudEvent) error
}

type PayoutService struct {
	publisher EventPublisher
	logger    *zap.Logger
}

func NewPayoutService(publisher EventPublisher, logger *zap.Logger) *PayoutService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PayoutService{publisher: publisher, logger: logger}
}

func (s *PayoutService) DisburseCredit(ctx context.Context, userID uuid.UUID, amountCents int64, redemptionID uuid.UUID) error {
	if userID == uuid.Nil {
		return domain.NewValidationError("user id is required")
	}
	if redemptionID == uuid.Nil {
		return domain.NewValidationError("redemption id is required")
	}
	if amountCents <= 0 {
		return domain.NewValidationError("amount must be positive")
	}

	event := events.CreditDisbursedEvent{
		RedemptionID: redemptionID,
		UserID:       userID,
		AmountCents:  amountCents,
		Currency:     "MYR",
		OccurredAt:   time.Now().UTC(),
	}
	cloudEvent, err := kafka.NewCloudEvent("service-payment", events.CreditDisbursed, event)
	if err != nil {
		return fmt.Errorf("create credit disbursed event: %w", err)
	}
	if err := s.publisher.PublishEvent(ctx, events.TopicPaymentEvents, cloudEvent); err != nil {
		return fmt.Errorf("publish credit disbursed event: %w", err)
	}
	s.logger.Info("loyalty credit disbursed",
		zap.String("redemption_id", redemptionID.String()),
		zap.String("user_id", userID.String()),
		zap.Int64("amount_cents", amountCents),
	)
	return nil
}
