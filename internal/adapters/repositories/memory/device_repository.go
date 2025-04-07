package memory

import (
	"context"
	"fmt"
	"sync"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/repositories"
)

// DeviceRepository is an in-memory implementation of the DeviceRepository interface
type DeviceRepository struct {
	devices      map[string]*device.Device // Primary index by ID
	typeIndex    map[string]map[string]*device.Device // Index by type -> ID -> Device
	statusIndex  map[device.Status]map[string]*device.Device // Index by status -> ID -> Device
	mu           sync.RWMutex
}

// NewDeviceRepository creates a new in-memory device repository
func NewDeviceRepository() *DeviceRepository {
	return &DeviceRepository{
		devices:      make(map[string]*device.Device),
		typeIndex:    make(map[string]map[string]*device.Device),
		statusIndex:  make(map[device.Status]map[string]*device.Device),
		mu:           sync.RWMutex{},
	}
}

// Register registers a new device
func (r *DeviceRepository) Register(ctx context.Context, dev *device.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if device already exists
	if _, exists := r.devices[dev.ID]; exists {
		builder := errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Device already exists: %s", dev.ID)).
			WithCode(errbuilder.CodeAlreadyExists)
		return builder
	}

	// Store device in primary index
	r.devices[dev.ID] = dev

	// Add to type index
	if _, exists := r.typeIndex[dev.Type]; !exists {
		r.typeIndex[dev.Type] = make(map[string]*device.Device)
	}
	r.typeIndex[dev.Type][dev.ID] = dev

	// Add to status index
	if _, exists := r.statusIndex[dev.Status]; !exists {
		r.statusIndex[dev.Status] = make(map[string]*device.Device)
	}
	r.statusIndex[dev.Status][dev.ID] = dev

	return nil
}

// Get retrieves a device by ID
func (r *DeviceRepository) Get(ctx context.Context, id string) (*device.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dev, exists := r.devices[id]
	if !exists {
		builder := errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Device not found: %s", id)).
			WithCode(errbuilder.CodeNotFound)
		return nil, builder
	}

	return dev, nil
}

// Update updates an existing device
func (r *DeviceRepository) Update(ctx context.Context, dev *device.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if device exists
	oldDev, exists := r.devices[dev.ID]
	if !exists {
		builder := errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Device not found: %s", dev.ID)).
			WithCode(errbuilder.CodeNotFound)
		return builder
	}

	// Remove from old indexes if type or status changed
	if oldDev.Type != dev.Type {
		delete(r.typeIndex[oldDev.Type], dev.ID)
		// Clean up empty maps
		if len(r.typeIndex[oldDev.Type]) == 0 {
			delete(r.typeIndex, oldDev.Type)
		}
	}

	if oldDev.Status != dev.Status {
		delete(r.statusIndex[oldDev.Status], dev.ID)
		// Clean up empty maps
		if len(r.statusIndex[oldDev.Status]) == 0 {
			delete(r.statusIndex, oldDev.Status)
		}
	}

	// Update device in primary index
	r.devices[dev.ID] = dev

	// Update type index
	if _, exists := r.typeIndex[dev.Type]; !exists {
		r.typeIndex[dev.Type] = make(map[string]*device.Device)
	}
	r.typeIndex[dev.Type][dev.ID] = dev

	// Update status index
	if _, exists := r.statusIndex[dev.Status]; !exists {
		r.statusIndex[dev.Status] = make(map[string]*device.Device)
	}
	r.statusIndex[dev.Status][dev.ID] = dev

	return nil
}

// Delete deletes a device by ID
func (r *DeviceRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if device exists
	dev, exists := r.devices[id]
	if !exists {
		builder := errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Device not found: %s", id)).
			WithCode(errbuilder.CodeNotFound)
		return builder
	}

	// Remove from type index
	delete(r.typeIndex[dev.Type], id)
	// Clean up empty maps
	if len(r.typeIndex[dev.Type]) == 0 {
		delete(r.typeIndex, dev.Type)
	}

	// Remove from status index
	delete(r.statusIndex[dev.Status], id)
	// Clean up empty maps
	if len(r.statusIndex[dev.Status]) == 0 {
		delete(r.statusIndex, dev.Status)
	}

	// Delete device from primary index
	delete(r.devices, id)
	return nil
}

// List lists devices with optional filtering
func (r *DeviceRepository) List(ctx context.Context, filter repositories.DeviceFilter) ([]*device.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Use a map to track unique devices that match all filters
	resultMap := make(map[string]*device.Device)

	// If we have type filters, use the type index for better performance
	if len(filter.Types) > 0 {
		for _, deviceType := range filter.Types {
			if devices, exists := r.typeIndex[deviceType]; exists {
				for id, dev := range devices {
					// Add to result map if not already there
					resultMap[id] = dev
				}
			}
		}
		// If no devices match the type filter, return empty result
		if len(resultMap) == 0 {
			return []*device.Device{}, nil
		}
	} else {
		// No type filter, add all devices to the result map
		for id, dev := range r.devices {
			resultMap[id] = dev
		}
		// Note: We can't use maps.Copy here because the map types are different (string->*Device vs string->interface{})
	}

	// If we have status filters, filter the result map
	if len(filter.Statuses) > 0 {
		// Create a temporary map to store devices that match the status filter
		tempMap := make(map[string]*device.Device)

		for _, status := range filter.Statuses {
			if devices, exists := r.statusIndex[status]; exists {
				for id, dev := range devices {
					// Only add if it's in the result map (passed the type filter)
					if _, ok := resultMap[id]; ok {
						tempMap[id] = dev
					}
				}
			}
		}

		// Replace result map with temp map
		resultMap = tempMap

		// If no devices match the status filter, return empty result
		if len(resultMap) == 0 {
			return []*device.Device{}, nil
		}
	}

	// Apply metadata filter
	if len(filter.Metadata) > 0 {
		// Create a list of devices to remove
		toRemove := make([]string, 0)

		for id, dev := range resultMap {
			metadataMatch := true
			for k, v := range filter.Metadata {
				if dev.Metadata[k] != v {
					metadataMatch = false
					break
				}
			}
			if !metadataMatch {
				toRemove = append(toRemove, id)
			}
		}

		// Remove devices that don't match metadata filter
		for _, id := range toRemove {
			delete(resultMap, id)
		}
	}

	// Convert map to slice for return
	result := make([]*device.Device, 0, len(resultMap))
	for _, dev := range resultMap {
		result = append(result, dev)
	}

	return result, nil
}
