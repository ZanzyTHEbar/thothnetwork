package memory

import (
	"context"
	"sync"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/room"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/repositories"
)

// RoomRepository is an in-memory implementation of the RoomRepository interface
type RoomRepository struct {
	rooms map[string]*room.Room
	mu    sync.RWMutex
}

// NewRoomRepository creates a new in-memory room repository
func NewRoomRepository() *RoomRepository {
	return &RoomRepository{
		rooms: make(map[string]*room.Room),
		mu:    sync.RWMutex{},
	}
}

// Create creates a new room
func (r *RoomRepository) Create(ctx context.Context, rm *room.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room already exists
	if _, exists := r.rooms[rm.ID]; exists {
		return errbuilder.New().
			WithMessage("Room already exists").
			WithField("room_id", rm.ID).
			Build()
	}

	// Store room
	r.rooms[rm.ID] = rm
	return nil
}

// Get retrieves a room by ID
func (r *RoomRepository) Get(ctx context.Context, id string) (*room.Room, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rm, exists := r.rooms[id]
	if !exists {
		return nil, errbuilder.New().
			WithMessage("Room not found").
			WithField("room_id", id).
			Build()
	}

	return rm, nil
}

// Update updates an existing room
func (r *RoomRepository) Update(ctx context.Context, rm *room.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room exists
	if _, exists := r.rooms[rm.ID]; !exists {
		return errbuilder.New().
			WithMessage("Room not found").
			WithField("room_id", rm.ID).
			Build()
	}

	// Update room
	r.rooms[rm.ID] = rm
	return nil
}

// Delete deletes a room by ID
func (r *RoomRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room exists
	if _, exists := r.rooms[id]; !exists {
		return errbuilder.New().
			WithMessage("Room not found").
			WithField("room_id", id).
			Build()
	}

	// Delete room
	delete(r.rooms, id)
	return nil
}

// List lists rooms with optional filtering
func (r *RoomRepository) List(ctx context.Context, filter repositories.RoomFilter) ([]*room.Room, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*room.Room, 0)

	for _, rm := range r.rooms {
		// Apply type filter
		if len(filter.Types) > 0 {
			typeMatch := false
			for _, t := range filter.Types {
				if rm.Type == t {
					typeMatch = true
					break
				}
			}
			if !typeMatch {
				continue
			}
		}

		// Apply device ID filter
		if filter.DeviceID != "" {
			deviceMatch := false
			for _, id := range rm.Devices {
				if id == filter.DeviceID {
					deviceMatch = true
					break
				}
			}
			if !deviceMatch {
				continue
			}
		}

		// Apply metadata filter
		if len(filter.Metadata) > 0 {
			metadataMatch := true
			for k, v := range filter.Metadata {
				if rm.Metadata[k] != v {
					metadataMatch = false
					break
				}
			}
			if !metadataMatch {
				continue
			}
		}

		// Add room to result
		result = append(result, rm)
	}

	return result, nil
}

// AddDevice adds a device to a room
func (r *RoomRepository) AddDevice(ctx context.Context, roomID, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room exists
	rm, exists := r.rooms[roomID]
	if !exists {
		return errbuilder.New().
			WithMessage("Room not found").
			WithField("room_id", roomID).
			Build()
	}

	// Add device to room
	if !rm.AddDevice(deviceID) {
		return errbuilder.New().
			WithMessage("Device already in room").
			WithField("room_id", roomID).
			WithField("device_id", deviceID).
			Build()
	}

	return nil
}

// RemoveDevice removes a device from a room
func (r *RoomRepository) RemoveDevice(ctx context.Context, roomID, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room exists
	rm, exists := r.rooms[roomID]
	if !exists {
		return errbuilder.New().
			WithMessage("Room not found").
			WithField("room_id", roomID).
			Build()
	}

	// Remove device from room
	if !rm.RemoveDevice(deviceID) {
		return errbuilder.New().
			WithMessage("Device not in room").
			WithField("room_id", roomID).
			WithField("device_id", deviceID).
			Build()
	}

	return nil
}

// ListDevices lists all devices in a room
func (r *RoomRepository) ListDevices(ctx context.Context, roomID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if room exists
	rm, exists := r.rooms[roomID]
	if !exists {
		return nil, errbuilder.New().
			WithMessage("Room not found").
			WithField("room_id", roomID).
			Build()
	}

	// Return a copy of the devices slice
	devices := make([]string, len(rm.Devices))
	copy(devices, rm.Devices)

	return devices, nil
}

// FindRoomsForDevice finds all rooms that contain a specific device
func (r *RoomRepository) FindRoomsForDevice(ctx context.Context, deviceID string) ([]*room.Room, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*room.Room, 0)

	for _, rm := range r.rooms {
		for _, id := range rm.Devices {
			if id == deviceID {
				result = append(result, rm)
				break
			}
		}
	}

	return result, nil
}
