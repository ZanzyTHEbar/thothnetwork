package twin

import (
	"time"
)

// DigitalTwin represents a virtual representation of a device
type DigitalTwin struct {
	// DeviceID is the ID of the device this twin represents
	DeviceID string `json:"device_id"`

	// State is the current state of the device
	State map[string]any `json:"state"`

	// Desired is the desired state of the device
	Desired map[string]any `json:"desired"`

	// Version is the version of the twin
	Version int64 `json:"version"`

	// LastUpdated is the timestamp when the twin was last updated
	LastUpdated time.Time `json:"last_updated"`

	// Metadata is additional information about the twin
	Metadata map[string]any `json:"metadata"`
}

// NewDigitalTwin creates a new digital twin for the given device ID
func NewDigitalTwin(deviceID string) *DigitalTwin {
	return &DigitalTwin{
		DeviceID:    deviceID,
		State:       make(map[string]any),
		Desired:     make(map[string]any),
		Version:     1,
		LastUpdated: time.Now(),
		Metadata:    make(map[string]any),
	}
}

// UpdateState updates the current state of the twin
func (t *DigitalTwin) UpdateState(state map[string]any) {
	t.State = state
	t.Version++
	t.LastUpdated = time.Now()
}

// UpdateDesired updates the desired state of the twin
func (t *DigitalTwin) UpdateDesired(desired map[string]any) {
	t.Desired = desired
	t.Version++
	t.LastUpdated = time.Now()
}

// SetMetadata sets a metadata value for the twin
func (t *DigitalTwin) SetMetadata(key string, value any) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]any)
	}
	t.Metadata[key] = value
	t.LastUpdated = time.Now()
}

// GetStateDelta returns the difference between desired and current state
func (t *DigitalTwin) GetStateDelta() map[string]any {
	delta := make(map[string]any)

	// Find keys in desired that are different or missing in state
	for key, desiredValue := range t.Desired {
		if stateValue, exists := t.State[key]; !exists || stateValue != desiredValue {
			delta[key] = desiredValue
		}
	}

	return delta
}
