package repositories

import (
	"context"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/twin"
)

// TwinRepository defines the interface for digital twin storage
type TwinRepository interface {
	// Get retrieves a digital twin by device ID
	Get(ctx context.Context, deviceID string) (*twin.DigitalTwin, error)

	// Update updates an existing digital twin
	Update(ctx context.Context, twin *twin.DigitalTwin) error

	// UpdateState updates just the state portion of a digital twin
	UpdateState(ctx context.Context, deviceID string, state map[string]any) error

	// UpdateDesired updates just the desired state portion of a digital twin
	UpdateDesired(ctx context.Context, deviceID string, desired map[string]any) error

	// Delete deletes a digital twin by device ID
	Delete(ctx context.Context, deviceID string) error
}
