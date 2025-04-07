package room

import (
	"slices"
	"time"
)

// Type represents the type of a room
type Type string

const (
	// TypeMany2Many allows many-to-many communication
	TypeMany2Many Type = "many2many"
	// TypeMany2One allows many-to-one communication
	TypeMany2One Type = "many2one"
	// TypeOne2Many allows one-to-many communication
	TypeOne2Many Type = "one2many"
)

// Room represents a logical grouping of devices
type Room struct {
	// ID is the unique identifier for the room
	ID string `json:"id"`

	// Name is the human-readable name of the room
	Name string `json:"name"`

	// Type is the type of the room
	Type Type `json:"type"`

	// Devices is the list of device IDs in the room
	Devices []string `json:"devices"`

	// deviceMap is a map of device IDs for O(1) lookups (not serialized)
	deviceMap map[string]struct{} `json:"-"`

	// Metadata is additional information about the room
	Metadata map[string]string `json:"metadata"`

	// CreatedAt is the timestamp when the room was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when the room was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// NewRoom creates a new room with the given ID and name
func NewRoom(id, name string, roomType Type) *Room {
	now := time.Now()
	return &Room{
		ID:        id,
		Name:      name,
		Type:      roomType,
		Devices:   make([]string, 0),
		deviceMap: make(map[string]struct{}),
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddDevice adds a device to the room
func (r *Room) AddDevice(deviceID string) bool {
	// Initialize deviceMap if nil
	if r.deviceMap == nil {
		r.deviceMap = make(map[string]struct{})
		for _, id := range r.Devices {
			r.deviceMap[id] = struct{}{}
		}
	}

	// Check if device is already in the room using O(1) lookup
	if _, exists := r.deviceMap[deviceID]; exists {
		return false
	}

	// Add device to the room
	r.Devices = append(r.Devices, deviceID)
	r.deviceMap[deviceID] = struct{}{}
	r.UpdatedAt = time.Now()
	return true
}

// RemoveDevice removes a device from the room
func (r *Room) RemoveDevice(deviceID string) bool {
	// Initialize deviceMap if nil
	if r.deviceMap == nil {
		r.deviceMap = make(map[string]struct{})
		for _, id := range r.Devices {
			r.deviceMap[id] = struct{}{}
		}
	}

	// Check if device exists in the room
	if _, exists := r.deviceMap[deviceID]; !exists {
		return false
	}

	// Remove device from the map
	delete(r.deviceMap, deviceID)

	// Remove device from the slice
	for i, id := range r.Devices {
		if id == deviceID {
			// Remove device from the room using slices.Delete
			r.Devices = slices.Delete(r.Devices, i, i+1)
			r.UpdatedAt = time.Now()
			return true
		}
	}

	// This should never happen if deviceMap is in sync with Devices
	return false
}

// SetMetadata sets a metadata value for the room
func (r *Room) SetMetadata(key, value string) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]string)
	}
	r.Metadata[key] = value
	r.UpdatedAt = time.Now()
}

// ContainsDevice checks if a device is in the room using O(1) lookup
func (r *Room) ContainsDevice(deviceID string) bool {
	// Initialize deviceMap if nil
	if r.deviceMap == nil {
		r.deviceMap = make(map[string]struct{})
		for _, id := range r.Devices {
			r.deviceMap[id] = struct{}{}
		}
	}

	_, exists := r.deviceMap[deviceID]
	return exists
}
