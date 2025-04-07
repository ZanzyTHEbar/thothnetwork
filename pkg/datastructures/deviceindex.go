package datastructures

import (
	"slices"
	"sync"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
)

// DeviceIndex is a specialized index for devices
type DeviceIndex struct {
	// Primary index by ID
	idIndex *BTree

	// Secondary indices
	typeIndex     map[string][]string // device type -> device IDs
	statusIndex   map[string][]string // device status -> device IDs
	locationIndex *SpatialIndex       // location -> device IDs

	// Mutex for thread safety
	mu sync.RWMutex
}

// NewDeviceIndex creates a new device index
func NewDeviceIndex() *DeviceIndex {
	return &DeviceIndex{
		idIndex:       NewBTree(8), // Degree 8 is a good balance for most use cases
		typeIndex:     make(map[string][]string),
		statusIndex:   make(map[string][]string),
		locationIndex: NewSpatialIndex(),
	}
}

// Add adds a device to the index
func (idx *DeviceIndex) Add(dev *device.Device) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Add to primary index
	idx.idIndex.Put(dev.ID, dev)

	// Add to type index
	idx.typeIndex[dev.Type] = append(idx.typeIndex[dev.Type], dev.ID)

	// Add to status index
	idx.statusIndex[string(dev.Status)] = append(idx.statusIndex[string(dev.Status)], dev.ID)

	// Add to location index if location is available
	if location, ok := dev.Metadata["location"]; ok {
		// Parse location and add to spatial index
		// This is a simplified example; in a real implementation,
		// you would parse the location string into coordinates
		idx.locationIndex.Add(dev.ID, location)
	}
}

// Update updates a device in the index
func (idx *DeviceIndex) Update(dev *device.Device) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Get the old device
	oldDevInterface, exists := idx.idIndex.Get(dev.ID)
	if !exists {
		// If the device doesn't exist, just add it
		idx.idIndex.Put(dev.ID, dev)
		idx.typeIndex[dev.Type] = append(idx.typeIndex[dev.Type], dev.ID)
		idx.statusIndex[string(dev.Status)] = append(idx.statusIndex[string(dev.Status)], dev.ID)
		return
	}

	oldDev := oldDevInterface.(*device.Device)

	// Remove from old indices if they've changed
	if oldDev.Type != dev.Type {
		idx.removeFromSlice(idx.typeIndex[oldDev.Type], dev.ID)
		idx.typeIndex[dev.Type] = append(idx.typeIndex[dev.Type], dev.ID)
	}

	if oldDev.Status != dev.Status {
		idx.removeFromSlice(idx.statusIndex[string(oldDev.Status)], dev.ID)
		idx.statusIndex[string(dev.Status)] = append(idx.statusIndex[string(dev.Status)], dev.ID)
	}

	// Update location if it's changed
	oldLocation, oldHasLocation := oldDev.Metadata["location"]
	newLocation, newHasLocation := dev.Metadata["location"]

	if oldHasLocation && (!newHasLocation || oldLocation != newLocation) {
		idx.locationIndex.Remove(dev.ID)
	}

	if newHasLocation && (!oldHasLocation || oldLocation != newLocation) {
		idx.locationIndex.Add(dev.ID, newLocation)
	}

	// Update primary index
	idx.idIndex.Put(dev.ID, dev)
}

// Remove removes a device from the index
func (idx *DeviceIndex) Remove(deviceID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Get the device
	devInterface, exists := idx.idIndex.Get(deviceID)
	if !exists {
		return
	}

	dev := devInterface.(*device.Device)

	// Remove from type index
	idx.removeFromSlice(idx.typeIndex[dev.Type], deviceID)

	// Remove from status index
	idx.removeFromSlice(idx.statusIndex[string(dev.Status)], deviceID)

	// Remove from location index
	idx.locationIndex.Remove(deviceID)

	// Remove from primary index
	idx.idIndex.Delete(deviceID)
}

// GetByID gets a device by ID
func (idx *DeviceIndex) GetByID(deviceID string) (*device.Device, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	devInterface, exists := idx.idIndex.Get(deviceID)
	if !exists {
		return nil, false
	}

	return devInterface.(*device.Device), true
}

// GetByType gets devices by type
func (idx *DeviceIndex) GetByType(deviceType string) []*device.Device {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	deviceIDs, exists := idx.typeIndex[deviceType]
	if !exists {
		return []*device.Device{}
	}

	devices := make([]*device.Device, 0, len(deviceIDs))
	for _, id := range deviceIDs {
		devInterface, exists := idx.idIndex.Get(id)
		if exists {
			devices = append(devices, devInterface.(*device.Device))
		}
	}

	return devices
}

// GetByStatus gets devices by status
func (idx *DeviceIndex) GetByStatus(status device.Status) []*device.Device {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	deviceIDs, exists := idx.statusIndex[string(status)]
	if !exists {
		return []*device.Device{}
	}

	devices := make([]*device.Device, 0, len(deviceIDs))
	for _, id := range deviceIDs {
		devInterface, exists := idx.idIndex.Get(id)
		if exists {
			devices = append(devices, devInterface.(*device.Device))
		}
	}

	return devices
}

// GetByLocation gets devices by location
func (idx *DeviceIndex) GetByLocation(location string, radiusMeters float64) []*device.Device {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	deviceIDs := idx.locationIndex.GetInRadius(location, radiusMeters)

	devices := make([]*device.Device, 0, len(deviceIDs))
	for _, id := range deviceIDs {
		devInterface, exists := idx.idIndex.Get(id)
		if exists {
			devices = append(devices, devInterface.(*device.Device))
		}
	}

	return devices
}

// GetAll gets all devices
func (idx *DeviceIndex) GetAll() []*device.Device {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	devices := make([]*device.Device, 0, idx.idIndex.Size())

	idx.idIndex.ForEach(func(_ string, value any) bool {
		devices = append(devices, value.(*device.Device))
		return true
	})

	return devices
}

// Count returns the number of devices in the index
func (idx *DeviceIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.idIndex.Size()
}

// Clear removes all devices from the index
func (idx *DeviceIndex) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.idIndex.Clear()
	idx.typeIndex = make(map[string][]string)
	idx.statusIndex = make(map[string][]string)
	idx.locationIndex = NewSpatialIndex()
}

// removeFromSlice removes a value from a slice
func (idx *DeviceIndex) removeFromSlice(slice []string, value string) []string {
	for i, v := range slice {
		if v == value {
			return slices.Delete(slice, i, i+1)
		}
	}
	return slice
}

// SpatialIndex is a simple spatial index
// In a real implementation, this would be a more sophisticated data structure
// like an R-tree or a geohash index
type SpatialIndex struct {
	locations map[string]string // deviceID -> location
	mu        sync.RWMutex
}

// NewSpatialIndex creates a new spatial index
func NewSpatialIndex() *SpatialIndex {
	return &SpatialIndex{
		locations: make(map[string]string),
	}
}

// Add adds a device to the spatial index
func (idx *SpatialIndex) Add(deviceID, location string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.locations[deviceID] = location
}

// Remove removes a device from the spatial index
func (idx *SpatialIndex) Remove(deviceID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.locations, deviceID)
}

// GetInRadius gets devices within a radius of a location
// This is a simplified implementation; in a real implementation,
// you would use a proper spatial index
func (idx *SpatialIndex) GetInRadius(location string, radiusMeters float64) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// In a real implementation, you would calculate the distance
	// between the given location and each device's location,
	// and return only those within the radius

	// For now, we'll just return all devices
	deviceIDs := make([]string, 0, len(idx.locations))
	for id := range idx.locations {
		deviceIDs = append(deviceIDs, id)
	}

	return deviceIDs
}
