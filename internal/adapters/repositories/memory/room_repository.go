package memory

import (
	"context"
	"fmt"
	"slices"
	"sync"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/room"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/repositories"
)

// RoomRepository is an in-memory implementation of the RoomRepository interface
type RoomRepository struct {
	rooms       map[string]*room.Room // Primary index by ID
	typeIndex   map[room.Type]map[string]*room.Room // Index by type -> ID -> Room
	deviceIndex map[string]map[string]*room.Room // Index by device ID -> room ID -> Room
	mu          sync.RWMutex
}

// NewRoomRepository creates a new in-memory room repository
func NewRoomRepository() *RoomRepository {
	return &RoomRepository{
		rooms:       make(map[string]*room.Room),
		typeIndex:   make(map[room.Type]map[string]*room.Room),
		deviceIndex: make(map[string]map[string]*room.Room),
		mu:          sync.RWMutex{},
	}
}

// Create creates a new room
func (r *RoomRepository) Create(ctx context.Context, rm *room.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room already exists
	if _, exists := r.rooms[rm.ID]; exists {
		return errbuilder.NewErrBuilder().
			WithMsg("Room already exists: " + rm.ID).
			WithCode(errbuilder.CodeAlreadyExists)
	}

	// Store room in primary index
	r.rooms[rm.ID] = rm

	// Add to type index
	if _, exists := r.typeIndex[rm.Type]; !exists {
		r.typeIndex[rm.Type] = make(map[string]*room.Room)
	}
	r.typeIndex[rm.Type][rm.ID] = rm

	// Add to device index for each device in the room
	for _, deviceID := range rm.Devices {
		if _, exists := r.deviceIndex[deviceID]; !exists {
			r.deviceIndex[deviceID] = make(map[string]*room.Room)
		}
		r.deviceIndex[deviceID][rm.ID] = rm
	}

	return nil
}

// Get retrieves a room by ID
func (r *RoomRepository) Get(ctx context.Context, id string) (*room.Room, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rm, exists := r.rooms[id]
	if !exists {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Room not found: " + id).
			WithCode(errbuilder.CodeNotFound)
	}

	return rm, nil
}

// Update updates an existing room
func (r *RoomRepository) Update(ctx context.Context, rm *room.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room exists
	oldRoom, exists := r.rooms[rm.ID]
	if !exists {
		return errbuilder.NewErrBuilder().
			WithMsg("Room not found: " + rm.ID).
			WithCode(errbuilder.CodeNotFound)
	}

	// Remove from old indexes if type changed
	if oldRoom.Type != rm.Type {
		delete(r.typeIndex[oldRoom.Type], rm.ID)
		// Clean up empty maps
		if len(r.typeIndex[oldRoom.Type]) == 0 {
			delete(r.typeIndex, oldRoom.Type)
		}
	}

	// Update device index
	// First, remove room from device index for devices that are no longer in the room
	for _, deviceID := range oldRoom.Devices {
		if !slices.Contains(rm.Devices, deviceID) {
			delete(r.deviceIndex[deviceID], rm.ID)
			// Clean up empty maps
			if len(r.deviceIndex[deviceID]) == 0 {
				delete(r.deviceIndex, deviceID)
			}
		}
	}

	// Add room to device index for new devices
	for _, deviceID := range rm.Devices {
		if !slices.Contains(oldRoom.Devices, deviceID) {
			if _, exists := r.deviceIndex[deviceID]; !exists {
				r.deviceIndex[deviceID] = make(map[string]*room.Room)
			}
			r.deviceIndex[deviceID][rm.ID] = rm
		}
	}

	// Update room in primary index
	r.rooms[rm.ID] = rm

	// Update type index
	if _, exists := r.typeIndex[rm.Type]; !exists {
		r.typeIndex[rm.Type] = make(map[string]*room.Room)
	}
	r.typeIndex[rm.Type][rm.ID] = rm

	return nil
}

// Delete deletes a room by ID
func (r *RoomRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room exists
	rm, exists := r.rooms[id]
	if !exists {
		return errbuilder.NewErrBuilder().
			WithMsg("Room not found: " + id).
			WithCode(errbuilder.CodeNotFound)
	}

	// Remove from type index
	delete(r.typeIndex[rm.Type], id)
	// Clean up empty maps
	if len(r.typeIndex[rm.Type]) == 0 {
		delete(r.typeIndex, rm.Type)
	}

	// Remove from device index for each device in the room
	for _, deviceID := range rm.Devices {
		delete(r.deviceIndex[deviceID], id)
		// Clean up empty maps
		if len(r.deviceIndex[deviceID]) == 0 {
			delete(r.deviceIndex, deviceID)
		}
	}

	// Delete room from primary index
	delete(r.rooms, id)
	return nil
}

// List lists rooms with optional filtering
func (r *RoomRepository) List(ctx context.Context, filter repositories.RoomFilter) ([]*room.Room, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Use a map to track unique rooms that match all filters
	resultMap := make(map[string]*room.Room)

	// If we have a device ID filter, use the device index for better performance
	if filter.DeviceID != "" {
		if rooms, exists := r.deviceIndex[filter.DeviceID]; exists {
			for id, rm := range rooms {
				resultMap[id] = rm
			}
		}
		// If no rooms match the device filter, return empty result
		if len(resultMap) == 0 {
			return []*room.Room{}, nil
		}
	} else if len(filter.Types) > 0 {
		// If we have type filters but no device filter, use the type index
		for _, roomType := range filter.Types {
			if rooms, exists := r.typeIndex[roomType]; exists {
				for id, rm := range rooms {
					resultMap[id] = rm
				}
			}
		}
		// If no rooms match the type filter, return empty result
		if len(resultMap) == 0 {
			return []*room.Room{}, nil
		}
	} else {
		// No device or type filter, add all rooms to the result map
		for id, rm := range r.rooms {
			resultMap[id] = rm
		}
	}

	// If we have both device and type filters, filter the result map by type
	if filter.DeviceID != "" && len(filter.Types) > 0 {
		// Create a list of rooms to remove
		toRemove := make([]string, 0)

		for id, rm := range resultMap {
			if !slices.Contains(filter.Types, rm.Type) {
				toRemove = append(toRemove, id)
			}
		}

		// Remove rooms that don't match type filter
		for _, id := range toRemove {
			delete(resultMap, id)
		}

		// If no rooms match both filters, return empty result
		if len(resultMap) == 0 {
			return []*room.Room{}, nil
		}
	}

	// Apply metadata filter
	if len(filter.Metadata) > 0 {
		// Create a list of rooms to remove
		toRemove := make([]string, 0)

		for id, rm := range resultMap {
			metadataMatch := true
			for k, v := range filter.Metadata {
				if rm.Metadata[k] != v {
					metadataMatch = false
					break
				}
			}
			if !metadataMatch {
				toRemove = append(toRemove, id)
			}
		}

		// Remove rooms that don't match metadata filter
		for _, id := range toRemove {
			delete(resultMap, id)
		}
	}

	// Convert map to slice for return
	result := make([]*room.Room, 0, len(resultMap))
	for _, rm := range resultMap {
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
		return errbuilder.NewErrBuilder().
			WithMsg("Room not found: " + roomID).
			WithCode(errbuilder.CodeNotFound)
	}

	// Add device to room
	if !rm.AddDevice(deviceID) {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Device %s already in room %s", deviceID, roomID)).
			WithCode(errbuilder.CodeAlreadyExists)
	}

	// Update device index
	if _, exists := r.deviceIndex[deviceID]; !exists {
		r.deviceIndex[deviceID] = make(map[string]*room.Room)
	}
	r.deviceIndex[deviceID][roomID] = rm

	return nil
}

// RemoveDevice removes a device from a room
func (r *RoomRepository) RemoveDevice(ctx context.Context, roomID, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if room exists
	rm, exists := r.rooms[roomID]
	if !exists {
		return errbuilder.NewErrBuilder().
			WithMsg("Room not found: " + roomID).
			WithCode(errbuilder.CodeNotFound)
	}

	// Remove device from room
	if !rm.RemoveDevice(deviceID) {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Device %s not in room %s", deviceID, roomID)).
			WithCode(errbuilder.CodeNotFound)
	}

	// Update device index
	if deviceRooms, exists := r.deviceIndex[deviceID]; exists {
		delete(deviceRooms, roomID)
		// Clean up empty maps
		if len(deviceRooms) == 0 {
			delete(r.deviceIndex, deviceID)
		}
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
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Room not found: " + roomID).
			WithCode(errbuilder.CodeNotFound)
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

	// Use the device index for better performance
	result := make([]*room.Room, 0)

	if rooms, exists := r.deviceIndex[deviceID]; exists {
		// Convert map to slice
		result = make([]*room.Room, 0, len(rooms))
		for _, rm := range rooms {
			result = append(result, rm)
		}
	}

	return result, nil
}
