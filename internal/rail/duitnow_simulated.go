package rail

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SimulatedRail is a development/testing implementation of Rail that simulates
// DuitNow FPX instant-transfer behaviour without touching a real payment network.
//
// Transfers always succeed: Submit sets the status to RailStatusProcessing and
// schedules a transition to RailStatusCompleted after the configured delay via
// the injected Clock.
//
// Context cancellation is honoured at the Submit call site only. Once the
// completion timer is scheduled the goroutine will run regardless — mirroring
// real-world async behaviour where you cannot "unsend" a bank transfer.
type SimulatedRail struct {
	delay  time.Duration
	logger *zap.Logger
	clock  Clock

	mu     sync.RWMutex
	states map[string]RailStatus
}

// NewSimulatedRail constructs a SimulatedRail.
//
//   - delay     — how long after Submit before the transfer is considered settled.
//   - logger    — structured logger for visibility into rail activity.
//   - clock     — Clock used to schedule the completion callback; inject a fake
//     clock in tests to avoid real sleeps.
func NewSimulatedRail(delay time.Duration, logger *zap.Logger, clock Clock) *SimulatedRail {
	return &SimulatedRail{
		delay:  delay,
		logger: logger,
		clock:  clock,
		states: make(map[string]RailStatus),
	}
}

// Submit initiates a simulated cash-out transfer. It immediately records the
// transfer as RailStatusProcessing and schedules a Clock-driven callback that
// transitions it to RailStatusCompleted after the configured delay.
//
// The returned txRef is stable and safe to persist; its format is "DN-<uuid>".
func (s *SimulatedRail) Submit(ctx context.Context, runnerID uuid.UUID, amountCents int64, destinationID uuid.UUID) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("rail: submit context cancelled: %w", err)
	}

	txRef := "DN-" + uuid.New().String()

	s.mu.Lock()
	s.states[txRef] = RailStatusProcessing
	s.mu.Unlock()

	s.logger.Info("[SIMULATED DUITNOW] transfer submitted",
		zap.String("tx_ref", txRef),
		zap.String("runner_id", runnerID.String()),
		zap.String("destination_id", destinationID.String()),
		zap.Int64("amount_cents", amountCents),
		zap.Duration("settle_in", s.delay),
	)

	// Schedule the completion transition. The callback runs in a new goroutine
	// (as per the Clock contract) so no mutex is held across the timer boundary.
	s.clock.AfterFunc(s.delay, func() {
		s.mu.Lock()
		s.states[txRef] = RailStatusCompleted
		s.mu.Unlock()

		s.logger.Info("[SIMULATED DUITNOW] transfer completed",
			zap.String("tx_ref", txRef),
		)
	})

	return txRef, nil
}

// Status returns the current RailStatus for the given txRef.
// Returns ErrRailRefNotFound if the txRef has never been submitted to this rail.
func (s *SimulatedRail) Status(ctx context.Context, txRef string) (RailStatus, error) {
	s.mu.RLock()
	st, ok := s.states[txRef]
	s.mu.RUnlock()

	if !ok {
		return "", ErrRailRefNotFound
	}
	return st, nil
}
