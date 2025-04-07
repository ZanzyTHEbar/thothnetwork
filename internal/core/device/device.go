package device

import (
	"time"
)

// Status represents the status of a device
type Status string

const (
	// StatusOnline indicates the device is online
	StatusOnline Status = "online"
	// StatusOffline indicates the device is offline
	StatusOffline Status = "offline"
	// StatusDisconnected indicates the device is disconnected
	StatusDisconnected Status = "disconnected"
	// StatusError indicates the device is in an error state
	StatusError Status = "error"
)

// Device represents an IoT device in the system
type Device struct {
	// ID is the unique identifier for the device
	ID string `json:"id"`

	// Name is the human-readable name of the device
	Name string `json:"name"`

	// Type is the type of the device
	Type string `json:"type"`

	// Status is the current status of the device
	Status Status `json:"status"`

	// Metadata is additional information about the device
	Metadata map[string]string `json:"metadata"`

	// LastSeen is the timestamp when the device was last seen
	LastSeen time.Time `json:"last_seen"`

	// CreatedAt is the timestamp when the device was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when the device was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// NewDevice creates a new device with the given ID and name
func NewDevice(id, name, deviceType string) *Device {
	now := time.Now()
	return &Device{
		ID:        id,
		Name:      name,
		Type:      deviceType,
		Status:    StatusOffline,
		Metadata:  make(map[string]string),
		LastSeen:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetStatus sets the status of the device
func (d *Device) SetStatus(status Status) {
	d.Status = status
	d.UpdatedAt = time.Now()
}

// SetMetadata sets a metadata value for the device
func (d *Device) SetMetadata(key, value string) {
	if d.Metadata == nil {
		d.Metadata = make(map[string]string)
	}
	d.Metadata[key] = value
	d.UpdatedAt = time.Now()
}

// UpdateLastSeen updates the last seen timestamp
func (d *Device) UpdateLastSeen() {
	now := time.Now()
	d.LastSeen = now
	d.UpdatedAt = now
}
