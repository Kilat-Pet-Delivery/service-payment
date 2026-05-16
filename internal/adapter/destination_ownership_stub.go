// Package adapter contains Anti-Corruption Layer adapters for external service integrations.
// This file provides a stub implementation of DestinationOwnership for development and testing.
// Replace when service-identity exposes payout destinations.
package adapter

import (
	"context"

	"github.com/google/uuid"
)

// DestinationOwnership is the interface for verifying that a payout destination
// belongs to a specific runner. The production implementation will call
// service-identity; the stub below is used until that endpoint exists.
type DestinationOwnership interface {
	// BelongsTo returns true if the given destination is owned by runnerID.
	BelongsTo(ctx context.Context, destinationID, runnerID uuid.UUID) (bool, error)
}

// InMemoryDestinationOwnership is an in-memory stub for DestinationOwnership.
// It holds a static map of destination UUID → owning runner UUID, seeded at
// construction time.
//
// Replace when service-identity exposes payout destinations.
type InMemoryDestinationOwnership struct {
	// ownerMap maps destinationID → runnerID
	ownerMap map[uuid.UUID]uuid.UUID
}

// NewInMemoryDestinationOwnership creates a stub with the given ownership map.
// Pass nil or an empty map to have all destinations return "not owned".
func NewInMemoryDestinationOwnership(ownerMap map[uuid.UUID]uuid.UUID) *InMemoryDestinationOwnership {
	if ownerMap == nil {
		ownerMap = make(map[uuid.UUID]uuid.UUID)
	}
	return &InMemoryDestinationOwnership{ownerMap: ownerMap}
}

// BelongsTo returns true if destinationID is mapped to runnerID in the stub map.
func (s *InMemoryDestinationOwnership) BelongsTo(_ context.Context, destinationID, runnerID uuid.UUID) (bool, error) {
	owner, ok := s.ownerMap[destinationID]
	if !ok {
		return false, nil
	}
	return owner == runnerID, nil
}
