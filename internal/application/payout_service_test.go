package application

import (
	"context"
	"testing"

	"github.com/Kilat-Pet-Delivery/lib-common/kafka"
	protoEvents "github.com/Kilat-Pet-Delivery/lib-proto/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_DisburseCredit_CreatesPayout_AndEmitsCreditDisbursed(t *testing.T) {
	publisher := &fakeEventPublisher{}
	service := NewPayoutService(publisher, nil)

	err := service.DisburseCredit(context.Background(), uuid.New(), 1250, uuid.New())
	require.NoError(t, err)
	require.Len(t, publisher.events, 1)
	require.Equal(t, protoEvents.TopicPaymentEvents, publisher.topic)
	require.Equal(t, protoEvents.CreditDisbursed, publisher.events[0].Type)
}

type fakeEventPublisher struct {
	topic  string
	events []kafka.CloudEvent
}

func (p *fakeEventPublisher) PublishEvent(_ context.Context, topic string, event kafka.CloudEvent) error {
	p.topic = topic
	p.events = append(p.events, event)
	return nil
}
