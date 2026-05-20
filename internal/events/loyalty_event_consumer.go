package events

import (
	"context"
	"strings"

	"github.com/Kilat-Pet-Delivery/lib-common/kafka"
	protoEvents "github.com/Kilat-Pet-Delivery/lib-proto/events"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type CreditDisburser interface {
	DisburseCredit(ctx context.Context, userID uuid.UUID, amountCents int64, redemptionID uuid.UUID) error
}

type LoyaltyEventConsumer struct {
	consumer *kafka.Consumer
	payouts  CreditDisburser
	logger   *zap.Logger
}

func NewLoyaltyEventConsumer(brokers []string, groupID string, payouts CreditDisburser, logger *zap.Logger) *LoyaltyEventConsumer {
	return &LoyaltyEventConsumer{
		consumer: kafka.NewConsumer(brokers, groupID, protoEvents.TopicLoyaltyEvents, logger),
		payouts:  payouts,
		logger:   logger,
	}
}

func (c *LoyaltyEventConsumer) Start(ctx context.Context) error {
	return c.consumer.Consume(ctx, c.handleMessage)
}

func (c *LoyaltyEventConsumer) Close() error {
	return c.consumer.Close()
}

func (c *LoyaltyEventConsumer) handleMessage(ctx context.Context, msg kafkago.Message) error {
	cloudEvent, err := kafka.ParseCloudEvent(msg.Value)
	if err != nil {
		c.logger.Error("failed to parse loyalty cloud event", zap.Error(err))
		return err
	}
	if !strings.EqualFold(cloudEvent.Type, protoEvents.RedemptionCreated) {
		return nil
	}

	var event protoEvents.RedemptionCreatedEvent
	if err := cloudEvent.ParseData(&event); err != nil {
		c.logger.Error("failed to parse RedemptionCreated event", zap.Error(err))
		return err
	}
	return c.payouts.DisburseCredit(ctx, event.UserID, event.Amount, event.RedemptionID)
}
