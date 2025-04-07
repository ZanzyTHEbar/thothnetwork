package repositories

import (
	"context"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
)

// DeviceFilter defines filters for listing devices
type DeviceFilter struct {
	Types    []string
	Statuses []device.Status
	Metadata map[string]string
}

// DeviceRepository defines the interface for device storage
type DeviceRepository interface {
	// Register registers a new device
	Register(ctx context.Context, device *device.Device) error

	// Get retrieves a device by ID
	Get(ctx context.Context, id string) (*device.Device, error)

	// Update updates an existing device
	Update(ctx context.Context, device *device.Device) error

	// Delete deletes a device by ID
	Delete(ctx context.Context, id string) error

	// List lists devices with optional filtering
	List(ctx context.Context, filter DeviceFilter) ([]*device.Device, error)
}
