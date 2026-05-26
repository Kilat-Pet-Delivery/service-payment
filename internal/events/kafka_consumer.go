package events

import (
	"context"
	"strings"

	"github.com/Kilat-Pet-Delivery/lib-common/kafka"
	"github.com/Kilat-Pet-Delivery/lib-proto/events"
	"github.com/Kilat-Pet-Delivery/service-payment/internal/application"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// BookingEventConsumer listens to booking events and triggers payment workflows.
type BookingEventConsumer struct {
	consumer       *kafka.Consumer
	paymentService *application.PaymentService
	walletService  *application.ShopWalletService
	logger         *zap.Logger
}

// NewBookingEventConsumer creates a new consumer for booking events.
func NewBookingEventConsumer(
	brokers []string,
	groupID string,
	paymentService *application.PaymentService,
	logger *zap.Logger,
	walletService ...*application.ShopWalletService,
) *BookingEventConsumer {
	consumer := kafka.NewConsumer(brokers, groupID, events.TopicBookingEvents, logger)
	var wallet *application.ShopWalletService
	if len(walletService) > 0 {
		wallet = walletService[0]
	}
	return &BookingEventConsumer{
		consumer:       consumer,
		paymentService: paymentService,
		walletService:  wallet,
		logger:         logger,
	}
}

// Start begins consuming booking events. It blocks until the context is cancelled.
func (c *BookingEventConsumer) Start(ctx context.Context) error {
	return c.consumer.Consume(ctx, c.handleMessage)
}

// handleMessage routes incoming Kafka messages to the appropriate handler.
func (c *BookingEventConsumer) handleMessage(ctx context.Context, msg kafkago.Message) error {
	cloudEvent, err := kafka.ParseCloudEvent(msg.Value)
	if err != nil {
		c.logger.Error("failed to parse cloud event from booking topic",
			zap.Error(err),
			zap.String("raw", string(msg.Value)),
		)
		return err
	}

	c.logger.Info("received booking event",
		zap.String("type", cloudEvent.Type),
		zap.String("id", cloudEvent.ID),
	)

	switch {
	case strings.EqualFold(cloudEvent.Type, events.BookingDeliveryConfirmed):
		return c.handleDeliveryConfirmed(ctx, cloudEvent)

	case strings.EqualFold(cloudEvent.Type, events.BookingCancelled):
		return c.handleBookingCancelled(ctx, cloudEvent)
	case strings.EqualFold(cloudEvent.Type, "booking.delivered"):
		return c.handleBookingDelivered(ctx, cloudEvent)

	default:
		c.logger.Debug("ignoring unhandled booking event type",
			zap.String("type", cloudEvent.Type),
		)
		return nil
	}
}

// handleBookingDelivered credits shop wallet ledger for shop bookings.
func (c *BookingEventConsumer) handleBookingDelivered(ctx context.Context, ce kafka.CloudEvent) error {
	if c.walletService == nil {
		return nil
	}
	var event struct {
		BookingID       uuid.UUID `json:"booking_id"`
		ShopID          uuid.UUID `json:"shop_id"`
		GrossSalesCents int64     `json:"gross_sales_cents"`
		NetSalesCents   int64     `json:"net_sales_cents"`
	}
	if err := ce.ParseData(&event); err != nil {
		return err
	}
	amount := event.NetSalesCents
	if amount == 0 {
		amount = event.GrossSalesCents
	}
	return c.walletService.HandleBookingDelivered(ctx, event.BookingID, event.ShopID, amount)
}

// handleDeliveryConfirmed processes a DeliveryConfirmedEvent.
func (c *BookingEventConsumer) handleDeliveryConfirmed(ctx context.Context, ce kafka.CloudEvent) error {
	var event events.DeliveryConfirmedEvent
	if err := ce.ParseData(&event); err != nil {
		c.logger.Error("failed to parse DeliveryConfirmedEvent data", zap.Error(err))
		return err
	}

	return c.paymentService.HandleDeliveryConfirmed(ctx, event)
}

// handleBookingCancelled processes a BookingCancelledEvent.
func (c *BookingEventConsumer) handleBookingCancelled(ctx context.Context, ce kafka.CloudEvent) error {
	var event events.BookingCancelledEvent
	if err := ce.ParseData(&event); err != nil {
		c.logger.Error("failed to parse BookingCancelledEvent data", zap.Error(err))
		return err
	}

	return c.paymentService.HandleBookingCancelled(ctx, event)
}

// Close closes the underlying Kafka consumer.
func (c *BookingEventConsumer) Close() error {
	return c.consumer.Close()
}
