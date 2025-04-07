package actor

import (
	"context"
	"fmt"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/anthdm/hollywood/actor"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/room"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/repositories"
	actorpkg "github.com/ZanzyTHEbar/thothnetwork/pkg/actor"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Service provides actor management functionality
type Service struct {
	actorSystem  *actorpkg.ActorSystem
	deviceRepo   repositories.DeviceRepository
	roomRepo     repositories.RoomRepository
	deviceActors map[string]*actor.PID
	roomActors   map[string]*actor.PID
	logger       logger.Logger
}

// NewService creates a new actor service
func NewService(
	actorSystem *actorpkg.ActorSystem,
	deviceRepo repositories.DeviceRepository,
	roomRepo repositories.RoomRepository,
	logger logger.Logger,
) *Service {
	return &Service{
		actorSystem:  actorSystem,
		deviceRepo:   deviceRepo,
		roomRepo:     roomRepo,
		deviceActors: make(map[string]*actor.PID),
		roomActors:   make(map[string]*actor.PID),
		logger:       logger.With("service", "actor"),
	}
}

// StartDeviceActor starts an actor for a device
func (s *Service) StartDeviceActor(ctx context.Context, deviceID string) error {
	// Check if device exists
	dev, err := s.deviceRepo.Get(ctx, deviceID)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to get device: %s", err.Error())).
			WithCode(errbuilder.CodeNotFound).
			WithCause(err)
	}

	// Check if actor already exists
	if _, exists := s.deviceActors[deviceID]; exists {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Actor for device %s already exists", deviceID)).
			WithCode(errbuilder.CodeAlreadyExists)
	}

	// Spawn device actor
	pid, err := s.actorSystem.SpawnDevice(deviceID)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to spawn device actor: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Store actor reference
	s.deviceActors[deviceID] = pid

	// Update device state
	s.logger.Info("Updating device actor state", "device_id", deviceID)
	_, err = s.actorSystem.Engine().Request(pid, &actorpkg.UpdateStateCommand{
		State: dev,
	}, nil)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to update device actor state: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	return nil
}

// StopDeviceActor stops an actor for a device
func (s *Service) StopDeviceActor(ctx context.Context, deviceID string) error {
	// Check if actor exists
	pid, exists := s.deviceActors[deviceID]
	if !exists {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Actor for device %s not found", deviceID)).
			WithCode(errbuilder.CodeNotFound)
	}

	// Stop actor
	s.actorSystem.Engine().Stop(pid)

	// Remove actor reference
	delete(s.deviceActors, deviceID)

	return nil
}

// StartRoomActor starts an actor for a room
func (s *Service) StartRoomActor(ctx context.Context, roomID string) error {
	// Check if room exists
	rm, err := s.roomRepo.Get(ctx, roomID)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to get room: %s", err.Error())).
			WithCode(errbuilder.CodeNotFound).
			WithCause(err)
	}

	// Check if actor already exists
	if _, exists := s.roomActors[roomID]; exists {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Actor for room %s already exists", roomID)).
			WithCode(errbuilder.CodeAlreadyExists)
	}

	// Spawn room actor
	pid, err := s.actorSystem.SpawnRoom(roomID)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to spawn room actor: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Store actor reference
	s.roomActors[roomID] = pid

	// Add devices to room
	for _, deviceID := range rm.Devices {
		// Get device actor
		devicePID, exists := s.deviceActors[deviceID]
		if !exists {
			// Start device actor if it doesn't exist
			err := s.StartDeviceActor(ctx, deviceID)
			if err != nil {
				s.logger.Warn("Failed to start device actor", "device_id", deviceID, "error", err)
				continue
			}
			devicePID = s.deviceActors[deviceID]
		}

		// Add device to room
		s.logger.Info("Adding device to room actor", "device_id", deviceID, "room_id", roomID)
		_, err := s.actorSystem.Engine().Request(pid, &actorpkg.AddDeviceCommand{
			DeviceID:  deviceID,
			DevicePID: devicePID,
		}, nil)
		if err != nil {
			s.logger.Warn("Failed to add device to room actor", "device_id", deviceID, "room_id", roomID, "error", err)
		}
	}

	return nil
}

// StopRoomActor stops an actor for a room
func (s *Service) StopRoomActor(ctx context.Context, roomID string) error {
	// Check if actor exists
	pid, exists := s.roomActors[roomID]
	if !exists {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Actor for room %s not found", roomID)).
			WithCode(errbuilder.CodeNotFound)
	}

	// Stop actor
	s.actorSystem.Engine().Stop(pid)

	// Remove actor reference
	delete(s.roomActors, roomID)

	return nil
}

// SendMessageToDevice sends a message to a device actor
func (s *Service) SendMessageToDevice(ctx context.Context, deviceID string, msg *message.Message) error {
	// Check if actor exists
	pid, exists := s.deviceActors[deviceID]
	if !exists {
		// Try to start device actor
		err := s.StartDeviceActor(ctx, deviceID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start device actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		pid = s.deviceActors[deviceID]
	}

	// Send message to device actor
	s.actorSystem.Engine().Send(pid, msg)

	return nil
}

// SendMessageToRoom sends a message to a room actor
func (s *Service) SendMessageToRoom(ctx context.Context, roomID string, msg *message.Message) error {
	// Check if actor exists
	pid, exists := s.roomActors[roomID]
	if !exists {
		// Try to start room actor
		err := s.StartRoomActor(ctx, roomID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start room actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		pid = s.roomActors[roomID]
	}

	// Send message to room actor
	s.actorSystem.Engine().Send(pid, msg)

	return nil
}

// GetDeviceState gets the state of a device actor
func (s *Service) GetDeviceState(ctx context.Context, deviceID string) (*device.Device, error) {
	// Check if actor exists
	pid, exists := s.deviceActors[deviceID]
	if !exists {
		// Try to start device actor
		err := s.StartDeviceActor(ctx, deviceID)
		if err != nil {
			return nil, errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start device actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		pid = s.deviceActors[deviceID]
	}

	// Get device state
	resp, err := s.actorSystem.Engine().Request(pid, &actorpkg.GetStateQuery{}, nil)
	if err != nil {
		return nil, errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to get device state: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Convert response to device state
	stateResp, ok := resp.(*actorpkg.GetStateResponse)
	if !ok {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Invalid response type").
			WithCode(errbuilder.CodeInternal)
	}

	return stateResp.State, nil
}

// GetRoomDevices gets the devices in a room actor
func (s *Service) GetRoomDevices(ctx context.Context, roomID string) ([]string, error) {
	// Check if actor exists
	pid, exists := s.roomActors[roomID]
	if !exists {
		// Try to start room actor
		err := s.StartRoomActor(ctx, roomID)
		if err != nil {
			return nil, errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start room actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		pid = s.roomActors[roomID]
	}

	// Get room devices
	resp, err := s.actorSystem.Engine().Request(pid, &actorpkg.GetDevicesQuery{}, nil)
	if err != nil {
		return nil, errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to get room devices: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Convert response to device IDs
	devicesResp, ok := resp.(*actorpkg.GetDevicesResponse)
	if !ok {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Invalid response type").
			WithCode(errbuilder.CodeInternal)
	}

	return devicesResp.DeviceIDs, nil
}

// AddDeviceToRoom adds a device to a room actor
func (s *Service) AddDeviceToRoom(ctx context.Context, roomID, deviceID string) error {
	// Check if room actor exists
	roomPID, exists := s.roomActors[roomID]
	if !exists {
		// Try to start room actor
		err := s.StartRoomActor(ctx, roomID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start room actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		roomPID = s.roomActors[roomID]
	}

	// Check if device actor exists
	devicePID, exists := s.deviceActors[deviceID]
	if !exists {
		// Try to start device actor
		err := s.StartDeviceActor(ctx, deviceID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start device actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		devicePID = s.deviceActors[deviceID]
	}

	// Add device to room in repository
	err := s.roomRepo.AddDevice(ctx, roomID, deviceID)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to add device to room in repository: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Add device to room actor
	_, err = s.actorSystem.Engine().Request(roomPID, &actorpkg.AddDeviceCommand{
		DeviceID:  deviceID,
		DevicePID: devicePID,
	}, nil)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to add device to room actor: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	return nil
}

// RemoveDeviceFromRoom removes a device from a room actor
func (s *Service) RemoveDeviceFromRoom(ctx context.Context, roomID, deviceID string) error {
	// Check if room actor exists
	roomPID, exists := s.roomActors[roomID]
	if !exists {
		// Try to start room actor
		err := s.StartRoomActor(ctx, roomID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start room actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		roomPID = s.roomActors[roomID]
	}

	// Remove device from room in repository
	err := s.roomRepo.RemoveDevice(ctx, roomID, deviceID)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to remove device from room in repository: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Remove device from room actor
	_, err = s.actorSystem.Engine().Request(roomPID, &actorpkg.RemoveDeviceCommand{
		DeviceID: deviceID,
	}, nil)
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to remove device from room actor: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	return nil
}

// StartAllDeviceActors starts actors for all devices
func (s *Service) StartAllDeviceActors(ctx context.Context) error {
	// Get all devices
	devices, err := s.deviceRepo.List(ctx, repositories.DeviceFilter{})
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to list devices: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Start actor for each device
	for _, dev := range devices {
		err := s.StartDeviceActor(ctx, dev.ID)
		if err != nil {
			s.logger.Warn("Failed to start device actor", "device_id", dev.ID, "error", err)
		}
	}

	return nil
}

// StartAllRoomActors starts actors for all rooms
func (s *Service) StartAllRoomActors(ctx context.Context) error {
	// Get all rooms
	rooms, err := s.roomRepo.List(ctx, repositories.RoomFilter{})
	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to list rooms: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Start actor for each room
	for _, rm := range rooms {
		err := s.StartRoomActor(ctx, rm.ID)
		if err != nil {
			s.logger.Warn("Failed to start room actor", "room_id", rm.ID, "error", err)
		}
	}

	return nil
}

// StopAllActors stops all actors
func (s *Service) StopAllActors() {
	// Stop all device actors
	for deviceID, pid := range s.deviceActors {
		s.logger.Info("Stopping device actor", "device_id", deviceID)
		s.actorSystem.Engine().Stop(pid)
	}
	s.deviceActors = make(map[string]*actor.PID)

	// Stop all room actors
	for roomID, pid := range s.roomActors {
		s.logger.Info("Stopping room actor", "room_id", roomID)
		s.actorSystem.Engine().Stop(pid)
	}
	s.roomActors = make(map[string]*actor.PID)
}
