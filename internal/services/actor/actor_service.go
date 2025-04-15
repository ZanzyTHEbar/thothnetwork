package actor

import (
	"context"
	"fmt"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/anthdm/hollywood/actor"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/room"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/repositories"
	actorpkg "github.com/ZanzyTHEbar/thothnetwork/pkg/actor"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/circuitbreaker"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/concurrent"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Service provides actor management functionality
type Service struct {
	actorSystem  *actorpkg.ActorSystem
	deviceRepo   repositories.DeviceRepository
	roomRepo     repositories.RoomRepository
	deviceActors *concurrent.HashMap
	roomActors   *concurrent.HashMap
	logger       logger.Logger

	// Circuit breakers for external dependencies
	deviceRepoCircuitBreaker *circuitbreaker.CircuitBreaker
	roomRepoCircuitBreaker   *circuitbreaker.CircuitBreaker

	// Add supervisor field to Service
	supervisor *actorpkg.Supervisor
}

// NewService creates a new actor service
func NewService(
	actorSystem *actorpkg.ActorSystem,
	deviceRepo repositories.DeviceRepository,
	roomRepo repositories.RoomRepository,
	logger logger.Logger,
) *Service {
	// Create logger
	serviceLogger := logger.With("service", "actor")

	// Create circuit breakers
	deviceRepoCircuitBreaker := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:             "device_repository",
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		Logger:           serviceLogger,
	})

	roomRepoCircuitBreaker := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:             "room_repository",
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		Logger:           serviceLogger,
	})

	// Create supervisor for device and room actors
	supervisor := actorpkg.CreateOneForOneSupervisor(10, time.Minute, serviceLogger)
	// Pass supervisor to actor system
	actorSystem.Supervisor = supervisor

	return &Service{
		actorSystem:              actorSystem,
		deviceRepo:               deviceRepo,
		roomRepo:                 roomRepo,
		deviceActors:             concurrent.NewHashMap(16),
		roomActors:               concurrent.NewHashMap(16),
		logger:                   serviceLogger,
		deviceRepoCircuitBreaker: deviceRepoCircuitBreaker,
		roomRepoCircuitBreaker:   roomRepoCircuitBreaker,
		supervisor:               supervisor,
	}
}

// StartDeviceActor starts an actor for a device
func (s *Service) StartDeviceActor(ctx context.Context, deviceID string) error {
	// Check if device exists with circuit breaker
	var dev *device.Device
	err := s.deviceRepoCircuitBreaker.Execute(func() error {
		var repoErr error
		dev, repoErr = s.deviceRepo.Get(ctx, deviceID)
		return repoErr
	})

	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to get device: %s", err.Error())).
			WithCode(errbuilder.CodeNotFound).
			WithCause(err)
	}

	// Check if actor already exists
	if _, exists := s.deviceActors.Get(deviceID); exists {
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
	s.deviceActors.Put(deviceID, pid)

	// Update device state
	s.logger.Info("Updating device actor state", "device_id", deviceID)
	_, err = s.actorSystem.Engine().Request(pid, &actorpkg.UpdateStateCommand{
		State: dev,
	}, 5*time.Second).Result()
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
	val, exists := s.deviceActors.Get(deviceID)
	if !exists {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Actor for device %s not found", deviceID)).
			WithCode(errbuilder.CodeNotFound)
	}
	pid := val.(*actor.PID)

	// Stop actor
	s.actorSystem.Engine().Stop(pid)

	// Remove actor reference
	s.deviceActors.Remove(deviceID)

	return nil
}

// StartRoomActor starts an actor for a room
func (s *Service) StartRoomActor(ctx context.Context, roomID string) error {
	// Check if room exists with circuit breaker
	var rm *room.Room
	err := s.roomRepoCircuitBreaker.Execute(func() error {
		var repoErr error
		rm, repoErr = s.roomRepo.Get(ctx, roomID)
		return repoErr
	})

	if err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to get room: %s", err.Error())).
			WithCode(errbuilder.CodeNotFound).
			WithCause(err)
	}

	// Check if actor already exists
	if _, exists := s.roomActors.Get(roomID); exists {
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
	s.roomActors.Put(roomID, pid)

	// Add devices to room
	for _, deviceID := range rm.Devices {
		// Get device actor
		val, exists := s.deviceActors.Get(deviceID)
		if !exists {
			// Start device actor if it doesn't exist
			err := s.StartDeviceActor(ctx, deviceID)
			if err != nil {
				s.logger.Warn("Failed to start device actor", "device_id", deviceID, "error", err)
				continue
			}
			val, _ = s.deviceActors.Get(deviceID)
		}
		devicePID := val.(*actor.PID)

		// Add device to room
		s.logger.Info("Adding device to room actor", "device_id", deviceID, "room_id", roomID)
		_, err := s.actorSystem.Engine().Request(pid, &actorpkg.AddDeviceCommand{
			DeviceID:  deviceID,
			DevicePID: devicePID,
		}, 5*time.Second).Result()
		if err != nil {
			s.logger.Warn("Failed to add device to room actor", "device_id", deviceID, "room_id", roomID, "error", err)
		}
	}

	return nil
}

// StopRoomActor stops an actor for a room
func (s *Service) StopRoomActor(ctx context.Context, roomID string) error {
	// Check if actor exists
	val, exists := s.roomActors.Get(roomID)
	if !exists {
		return errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Actor for room %s not found", roomID)).
			WithCode(errbuilder.CodeNotFound)
	}
	pid := val.(*actor.PID)

	// Stop actor
	s.actorSystem.Engine().Stop(pid)

	// Remove actor reference
	s.roomActors.Remove(roomID)

	return nil
}

// SendMessageToDevice sends a message to a device actor
func (s *Service) SendMessageToDevice(ctx context.Context, deviceID string, msg *message.Message) error {
	// Check if actor exists
	val, exists := s.deviceActors.Get(deviceID)
	if !exists {
		// Try to start device actor
		err := s.StartDeviceActor(ctx, deviceID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start device actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		val, _ = s.deviceActors.Get(deviceID)
	}
	pid := val.(*actor.PID)

	// Send message to device actor
	s.actorSystem.Engine().Send(pid, msg)

	return nil
}

// SendMessageToRoom sends a message to a room actor
func (s *Service) SendMessageToRoom(ctx context.Context, roomID string, msg *message.Message) error {
	// Check if actor exists
	val, exists := s.roomActors.Get(roomID)
	if !exists {
		// Try to start room actor
		err := s.StartRoomActor(ctx, roomID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start room actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		val, _ = s.roomActors.Get(roomID)
	}
	pid := val.(*actor.PID)

	// Send message to room actor
	s.actorSystem.Engine().Send(pid, msg)

	return nil
}

// GetDeviceState gets the state of a device actor
func (s *Service) GetDeviceState(ctx context.Context, deviceID string) (*device.Device, error) {
	// Check if actor exists
	val, exists := s.deviceActors.Get(deviceID)
	if !exists {
		// Try to start device actor
		err := s.StartDeviceActor(ctx, deviceID)
		if err != nil {
			return nil, errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start device actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		val, _ = s.deviceActors.Get(deviceID)
	}
	pid := val.(*actor.PID)

	// Get device state
	response, err := s.actorSystem.Engine().Request(pid, &actorpkg.GetStateQuery{}, 5*time.Second).Result()
	if err != nil {
		return nil, errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to get device state: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Convert response to device state
	stateResp, ok := response.(*actorpkg.GetStateResponse)
	if !ok {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Invalid response type").
			WithCode(errbuilder.CodeInternal)
	}

	return stateResp.State, nil
}

// GetRoomDevices gets the devices in a room actor
func (s *Service) GetRoomDevices(ctx context.Context, roomID string) (map[string]*actor.PID, error) {
	// Check if actor exists
	val, exists := s.roomActors.Get(roomID)
	if !exists {
		// Try to start room actor
		err := s.StartRoomActor(ctx, roomID)
		if err != nil {
			return nil, errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start room actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		val, _ = s.roomActors.Get(roomID)
	}
	pid := val.(*actor.PID)

	// Get room devices
	response, err := s.actorSystem.Engine().
		Request(pid, &actorpkg.GetDevicesQuery{}, 5*time.Second).
		Result()
	if err != nil {
		return nil, errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to get room devices: %s", err.Error())).
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Convert response to device IDs
	devicesResp, ok := response.(*actorpkg.GetDevicesResponse)
	if !ok {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Invalid response type").
			WithCode(errbuilder.CodeInternal)
	}

	return devicesResp.Devices, nil
}

// AddDeviceToRoom adds a device to a room actor
func (s *Service) AddDeviceToRoom(ctx context.Context, roomID, deviceID string) error {
	// Check if room actor exists
	roomVal, roomExists := s.roomActors.Get(roomID)
	if !roomExists {
		// Try to start room actor
		err := s.StartRoomActor(ctx, roomID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start room actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		roomVal, _ = s.roomActors.Get(roomID)
	}
	roomPID := roomVal.(*actor.PID)

	// Check if device actor exists
	deviceVal, deviceExists := s.deviceActors.Get(deviceID)
	if !deviceExists {
		// Try to start device actor
		err := s.StartDeviceActor(ctx, deviceID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start device actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		deviceVal, _ = s.deviceActors.Get(deviceID)
	}
	devicePID := deviceVal.(*actor.PID)

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
	}, 5*time.Second).Result()
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
	roomVal, roomExists := s.roomActors.Get(roomID)
	if !roomExists {
		// Try to start room actor
		err := s.StartRoomActor(ctx, roomID)
		if err != nil {
			return errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to start room actor: %s", err.Error())).
				WithCode(errbuilder.CodeNotFound).
				WithCause(err)
		}
		roomVal, _ = s.roomActors.Get(roomID)
	}
	roomPID := roomVal.(*actor.PID)

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
	}, 5*time.Second).Result()
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
	s.deviceActors.ForEach(func(deviceID string, val interface{}) bool {
		pid := val.(*actor.PID)
		s.logger.Info("Stopping device actor", "device_id", deviceID)
		s.actorSystem.Engine().Stop(pid)
		return true
	})
	s.deviceActors = concurrent.NewHashMap(16)

	// Stop all room actors
	s.roomActors.ForEach(func(roomID string, val interface{}) bool {
		pid := val.(*actor.PID)
		s.logger.Info("Stopping room actor", "room_id", roomID)
		s.actorSystem.Engine().Stop(pid)
		return true
	})
	s.roomActors = concurrent.NewHashMap(16)
}
