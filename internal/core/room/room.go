package room

import (
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
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddDevice adds a device to the room
func (r *Room) AddDevice(deviceID string) bool {
	// Check if device is already in the room
	for _, id := range r.Devices {
		if id == deviceID {
			return false
		}
	}
	
	// Add device to the room
	r.Devices = append(r.Devices, deviceID)
	r.UpdatedAt = time.Now()
	return true
}

// RemoveDevice removes a device from the room
func (r *Room) RemoveDevice(deviceID string) bool {
	for i, id := range r.Devices {
		if id == deviceID {
			// Remove device from the room
			r.Devices = append(r.Devices[:i], r.Devices[i+1:]...)
			r.UpdatedAt = time.Now()
			return true
		}
	}
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
