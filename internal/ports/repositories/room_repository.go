package repositories

import (
	"context"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/room"
)

// RoomFilter defines filters for listing rooms
type RoomFilter struct {
	Types    []room.Type
	DeviceID string
	Metadata map[string]string
}

// RoomRepository defines the interface for room storage
type RoomRepository interface {
	// Create creates a new room
	Create(ctx context.Context, room *room.Room) error

	// Get retrieves a room by ID
	Get(ctx context.Context, id string) (*room.Room, error)

	// Update updates an existing room
	Update(ctx context.Context, room *room.Room) error

	// Delete deletes a room by ID
	Delete(ctx context.Context, id string) error

	// List lists rooms with optional filtering
	List(ctx context.Context, filter RoomFilter) ([]*room.Room, error)

	// AddDevice adds a device to a room
	AddDevice(ctx context.Context, roomID, deviceID string) error

	// RemoveDevice removes a device from a room
	RemoveDevice(ctx context.Context, roomID, deviceID string) error

	// ListDevices lists all devices in a room
	ListDevices(ctx context.Context, roomID string) ([]string, error)

	// FindRoomsForDevice finds all rooms that contain a specific device
	FindRoomsForDevice(ctx context.Context, deviceID string) ([]*room.Room, error)
}
