package room

import (
	"context"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/room"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/brokers"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/repositories"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Service provides room management functionality
type Service struct {
	roomRepo      repositories.RoomRepository
	deviceRepo    repositories.DeviceRepository
	messageBroker brokers.MessageBroker
	logger        logger.Logger
}

// NewService creates a new room service
func NewService(
	roomRepo repositories.RoomRepository,
	deviceRepo repositories.DeviceRepository,
	messageBroker brokers.MessageBroker,
	logger logger.Logger,
) *Service {
	return &Service{
		roomRepo:      roomRepo,
		deviceRepo:    deviceRepo,
		messageBroker: messageBroker,
		logger:        logger.With("service", "room"),
	}
}

// CreateRoom creates a new room
func (s *Service) CreateRoom(ctx context.Context, r *room.Room) (string, error) {
	// Generate ID if not provided
	if r.ID == "" {
		r.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	// Create room
	if err := s.roomRepo.Create(ctx, r); err != nil {
		return "", errbuilder.GenericErr("Failed to create room", err)
	}

	// Publish room created event
	event := message.New(
		uuid.New().String(),
		"system",
		"*",
		message.TypeEvent,
		[]byte(`{"event":"room_created","room_id":"`+r.ID+`"}`),
	)
	event.SetContentType("application/json")

	if err := s.messageBroker.Publish(ctx, "events.room.created", event); err != nil {
		s.logger.Warn("Failed to publish room created event", "error", err)
	}

	return r.ID, nil
}

// GetRoom retrieves a room by ID
func (s *Service) GetRoom(ctx context.Context, id string) (*room.Room, error) {
	r, err := s.roomRepo.Get(ctx, id)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to get room", err)
	}
	return r, nil
}

// UpdateRoom updates an existing room
func (s *Service) UpdateRoom(ctx context.Context, r *room.Room) error {
	// Update timestamp
	r.UpdatedAt = time.Now()

	if err := s.roomRepo.Update(ctx, r); err != nil {
		return errbuilder.GenericErr("Failed to update room", err)
	}

	// Publish room updated event
	event := message.New(
		uuid.New().String(),
		"system",
		"*",
		message.TypeEvent,
		[]byte(`{"event":"room_updated","room_id":"`+r.ID+`"}`),
	)
	event.SetContentType("application/json")

	if err := s.messageBroker.Publish(ctx, "events.room.updated", event); err != nil {
		s.logger.Warn("Failed to publish room updated event", "error", err)
	}

	return nil
}

// DeleteRoom deletes a room
func (s *Service) DeleteRoom(ctx context.Context, id string) error {
	if err := s.roomRepo.Delete(ctx, id); err != nil {
		return errbuilder.GenericErr("Failed to delete room", err)
	}

	// Publish room deleted event
	event := message.New(
		uuid.New().String(),
		"system",
		"*",
		message.TypeEvent,
		[]byte(`{"event":"room_deleted","room_id":"`+id+`"}`),
	)
	event.SetContentType("application/json")

	if err := s.messageBroker.Publish(ctx, "events.room.deleted", event); err != nil {
		s.logger.Warn("Failed to publish room deleted event", "error", err)
	}

	return nil
}

// ListRooms lists rooms with optional filtering
func (s *Service) ListRooms(ctx context.Context, filter repositories.RoomFilter) ([]*room.Room, error) {
	rooms, err := s.roomRepo.List(ctx, filter)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to list rooms", err)
	}
	return rooms, nil
}

// AddDeviceToRoom adds a device to a room
func (s *Service) AddDeviceToRoom(ctx context.Context, roomID, deviceID string) error {
	// Check if device exists
	_, err := s.deviceRepo.Get(ctx, deviceID)
	if err != nil {
		return errbuilder.GenericErr("Device not found", err)
	}

	// Add device to room
	if err := s.roomRepo.AddDevice(ctx, roomID, deviceID); err != nil {
		return errbuilder.GenericErr("Failed to add device to room", err)
	}

	// Publish device added to room event
	event := message.New(
		uuid.New().String(),
		"system",
		"*",
		message.TypeEvent,
		[]byte(`{"event":"device_added_to_room","room_id":"`+roomID+`","device_id":"`+deviceID+`"}`),
	)
	event.SetContentType("application/json")

	if err := s.messageBroker.Publish(ctx, "events.room.device_added", event); err != nil {
		s.logger.Warn("Failed to publish device added to room event", "error", err)
	}

	return nil
}

// RemoveDeviceFromRoom removes a device from a room
func (s *Service) RemoveDeviceFromRoom(ctx context.Context, roomID, deviceID string) error {
	if err := s.roomRepo.RemoveDevice(ctx, roomID, deviceID); err != nil {
		return errbuilder.GenericErr("Failed to remove device from room", err)
	}

	// Publish device removed from room event
	event := message.New(
		uuid.New().String(),
		"system",
		"*",
		message.TypeEvent,
		[]byte(`{"event":"device_removed_from_room","room_id":"`+roomID+`","device_id":"`+deviceID+`"}`),
	)
	event.SetContentType("application/json")

	if err := s.messageBroker.Publish(ctx, "events.room.device_removed", event); err != nil {
		s.logger.Warn("Failed to publish device removed from room event", "error", err)
	}

	return nil
}

// ListDevicesInRoom lists all devices in a room
func (s *Service) ListDevicesInRoom(ctx context.Context, roomID string) ([]string, error) {
	devices, err := s.roomRepo.ListDevices(ctx, roomID)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to list devices in room", err)
	}
	return devices, nil
}

// PublishToRoom publishes a message to all devices in a room
func (s *Service) PublishToRoom(ctx context.Context, roomID string, msg *message.Message) error {
	// Get the room to check its type
	r, err := s.roomRepo.Get(ctx, roomID)
	if err != nil {
		return errbuilder.GenericErr("Failed to get room", err)
	}

	// Check room type and message source
	if r.Type == room.TypeMany2One && msg.Source != r.Devices[0] {
		return errbuilder.GenericErr("Only the primary device can publish to a many-to-one room", nil)
	}

	if r.Type == room.TypeOne2Many && msg.Target != r.Devices[0] {
		return errbuilder.GenericErr("Messages in a one-to-many room must target the primary device", nil)
	}

	// Publish message to room topic
	if err := s.messageBroker.Publish(ctx, "rooms."+roomID, msg); err != nil {
		return errbuilder.GenericErr("Failed to publish message to room", err)
	}

	return nil
}

// FindRoomsForDevice finds all rooms that contain a specific device
func (s *Service) FindRoomsForDevice(ctx context.Context, deviceID string) ([]*room.Room, error) {
	rooms, err := s.roomRepo.FindRoomsForDevice(ctx, deviceID)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to find rooms for device", err)
	}
	return rooms, nil
}

