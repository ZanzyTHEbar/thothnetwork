package memory

import (
	"context"
	"sync"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/repositories"
)

// DeviceRepository is an in-memory implementation of the DeviceRepository interface
type DeviceRepository struct {
	devices map[string]*device.Device
	mu      sync.RWMutex
}

// NewDeviceRepository creates a new in-memory device repository
func NewDeviceRepository() *DeviceRepository {
	return &DeviceRepository{
		devices: make(map[string]*device.Device),
		mu:      sync.RWMutex{},
	}
}

// Register registers a new device
func (r *DeviceRepository) Register(ctx context.Context, dev *device.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if device already exists
	if _, exists := r.devices[dev.ID]; exists {
		return errbuilder.New().
			WithMessage("Device already exists").
			WithField("device_id", dev.ID).
			Build()
	}

	// Store device
	r.devices[dev.ID] = dev
	return nil
}

// Get retrieves a device by ID
func (r *DeviceRepository) Get(ctx context.Context, id string) (*device.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dev, exists := r.devices[id]
	if !exists {
		return nil, errbuilder.New().
			WithMessage("Device not found").
			WithField("device_id", id).
			Build()
	}

	return dev, nil
}

// Update updates an existing device
func (r *DeviceRepository) Update(ctx context.Context, dev *device.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if device exists
	if _, exists := r.devices[dev.ID]; !exists {
		return errbuilder.New().
			WithMessage("Device not found").
			WithField("device_id", dev.ID).
			Build()
	}

	// Update device
	r.devices[dev.ID] = dev
	return nil
}

// Delete deletes a device by ID
func (r *DeviceRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if device exists
	if _, exists := r.devices[id]; !exists {
		return errbuilder.New().
			WithMessage("Device not found").
			WithField("device_id", id).
			Build()
	}

	// Delete device
	delete(r.devices, id)
	return nil
}

// List lists devices with optional filtering
func (r *DeviceRepository) List(ctx context.Context, filter repositories.DeviceFilter) ([]*device.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*device.Device, 0)

	for _, dev := range r.devices {
		// Apply type filter
		if len(filter.Types) > 0 {
			typeMatch := false
			for _, t := range filter.Types {
				if dev.Type == t {
					typeMatch = true
					break
				}
			}
			if !typeMatch {
				continue
			}
		}

		// Apply status filter
		if len(filter.Statuses) > 0 {
			statusMatch := false
			for _, s := range filter.Statuses {
				if dev.Status == s {
					statusMatch = true
					break
				}
			}
			if !statusMatch {
				continue
			}
		}

		// Apply metadata filter
		if len(filter.Metadata) > 0 {
			metadataMatch := true
			for k, v := range filter.Metadata {
				if dev.Metadata[k] != v {
					metadataMatch = false
					break
				}
			}
			if !metadataMatch {
				continue
			}
		}

		// Add device to result
		result = append(result, dev)
	}

	return result, nil
}
