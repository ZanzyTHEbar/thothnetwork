package memory

import (
	"context"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/twin"
)

// TwinRepository is an in-memory implementation of the TwinRepository interface
type TwinRepository struct {
	twins map[string]*twin.DigitalTwin
	mu    sync.RWMutex
}

// NewTwinRepository creates a new in-memory twin repository
func NewTwinRepository() *TwinRepository {
	return &TwinRepository{
		twins: make(map[string]*twin.DigitalTwin),
		mu:    sync.RWMutex{},
	}
}

// Get retrieves a digital twin by device ID
func (r *TwinRepository) Get(ctx context.Context, deviceID string) (*twin.DigitalTwin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dt, exists := r.twins[deviceID]
	if !exists {
		return nil, errbuilder.GenericErr("Digital twin not found", nil)
	}

	return dt, nil
}

// Update updates an existing digital twin
func (r *TwinRepository) Update(ctx context.Context, dt *twin.DigitalTwin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Store twin
	r.twins[dt.DeviceID] = dt
	return nil
}

// UpdateState updates just the state portion of a digital twin
func (r *TwinRepository) UpdateState(ctx context.Context, deviceID string, state map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if twin exists
	dt, exists := r.twins[deviceID]
	if !exists {
		return errbuilder.GenericErr("Digital twin not found", nil)
	}

	// Update state
	dt.State = state
	dt.Version++
	dt.LastUpdated = time.Now()

	return nil
}

// UpdateDesired updates just the desired state portion of a digital twin
func (r *TwinRepository) UpdateDesired(ctx context.Context, deviceID string, desired map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if twin exists
	dt, exists := r.twins[deviceID]
	if !exists {
		return errbuilder.GenericErr("Digital twin not found", nil)
	}

	// Update desired state
	dt.Desired = desired
	dt.Version++
	dt.LastUpdated = time.Now()

	return nil
}

// Delete deletes a digital twin by device ID
func (r *TwinRepository) Delete(ctx context.Context, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if twin exists
	if _, exists := r.twins[deviceID]; !exists {
		return errbuilder.GenericErr("Digital twin not found", nil)
	}

	// Delete twin
	delete(r.twins, deviceID)
	return nil
}
