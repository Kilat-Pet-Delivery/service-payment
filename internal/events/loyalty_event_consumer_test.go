package events

import (
	"context"
	"encoding/json"
	"testing"

	commonKafka "github.com/Kilat-Pet-Delivery/lib-common/kafka"
	protoEvents "github.com/Kilat-Pet-Delivery/lib-proto/events"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func Test_RedemptionCreated_TriggersPayout(t *testing.T) {
	payouts := &fakePayouts{}
	consumer := &LoyaltyEventConsumer{payouts: payouts, logger: zap.NewNop()}
	event := protoEvents.RedemptionCreatedEvent{
		RedemptionID: uuid.New(),
		UserID:       uuid.New(),
		Amount:       1200,
		Currency:     "MYR",
	}
	cloudEvent, err := commonKafka.NewCloudEvent("test", protoEvents.RedemptionCreated, event)
	require.NoError(t, err)

	raw, err := json.Marshal(cloudEvent)
	require.NoError(t, err)
	require.NoError(t, consumer.handleMessage(context.Background(), kafkago.Message{Value: raw}))

	require.Equal(t, event.RedemptionID, payouts.redemptionID)
	require.Equal(t, event.UserID, payouts.userID)
	require.Equal(t, event.Amount, payouts.amountCents)
}

type fakePayouts struct {
	userID       uuid.UUID
	amountCents  int64
	redemptionID uuid.UUID
}

func (p *fakePayouts) DisburseCredit(_ context.Context, userID uuid.UUID, amountCents int64, redemptionID uuid.UUID) error {
	p.userID = userID
	p.amountCents = amountCents
	p.redemptionID = redemptionID
	return nil
}
