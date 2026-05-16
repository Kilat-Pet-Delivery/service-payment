package rail

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// RailStatus represents the status of a payment rail transaction.
type RailStatus string

const (
	// RailStatusProcessing means the transfer has been submitted and is in-flight.
	RailStatusProcessing RailStatus = "PROCESSING"
	// RailStatusCompleted means the transfer settled successfully.
	RailStatusCompleted RailStatus = "COMPLETED"
	// RailStatusFailed means the transfer was rejected or could not settle.
	RailStatusFailed RailStatus = "FAILED"
)

// ErrRailRefNotFound is returned by Status when the txRef is not known to this rail.
var ErrRailRefNotFound = errors.New("rail: txRef not found")

// Rail is the interface for outbound cash-out payment rails.
// The production DuitNow integration is a drop-in replacement for SimulatedRail.
type Rail interface {
	// Submit initiates a cash-out transfer. It returns a txRef the caller must
	// persist; the transfer completes asynchronously (poll Status to observe).
	Submit(ctx context.Context, runnerID uuid.UUID, amountCents int64, destinationID uuid.UUID) (txRef string, err error)

	// Status returns the current status for a previously submitted txRef.
	// Returns ErrRailRefNotFound if the txRef is unknown.
	Status(ctx context.Context, txRef string) (RailStatus, error)
}

// Clock abstracts time scheduling so tests can drive transitions synchronously.
type Clock interface {
	// AfterFunc schedules f to be called in a new goroutine after duration d.
	AfterFunc(d time.Duration, f func())
}

// RealClock is the production Clock backed by time.AfterFunc.
type RealClock struct{}

// AfterFunc delegates to the standard library time.AfterFunc.
func (RealClock) AfterFunc(d time.Duration, f func()) {
	time.AfterFunc(d, f)
}
