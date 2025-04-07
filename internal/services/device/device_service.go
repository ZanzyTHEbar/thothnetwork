package device

import (
	"context"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/twin"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/brokers"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/repositories"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Service provides device management functionality
type Service struct {
	deviceRepo    repositories.DeviceRepository
	twinRepo      repositories.TwinRepository
	messageBroker brokers.MessageBroker
	logger        logger.Logger
}

// NewService creates a new device service
func NewService(
	deviceRepo repositories.DeviceRepository,
	twinRepo repositories.TwinRepository,
	messageBroker brokers.MessageBroker,
	logger logger.Logger,
) *Service {
	return &Service{
		deviceRepo:    deviceRepo,
		twinRepo:      twinRepo,
		messageBroker: messageBroker,
		logger:        logger.With("service", "device"),
	}
}

// RegisterDevice registers a new device and creates its digital twin
func (s *Service) RegisterDevice(ctx context.Context, dev *device.Device) (string, error) {
	// Generate ID if not provided
	if dev.ID == "" {
		dev.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	dev.CreatedAt = now
	dev.UpdatedAt = now
	dev.LastSeen = now

	// Register device
	if err := s.deviceRepo.Register(ctx, dev); err != nil {
		return "", errbuilder.GenericErr("Failed to register device", err)
	}

	// Create digital twin
	dt := twin.NewDigitalTwin(dev.ID)
	if err := s.twinRepo.Update(ctx, dt); err != nil {
		// Try to clean up the device if twin creation fails
		_ = s.deviceRepo.Delete(ctx, dev.ID)
		return "", errbuilder.GenericErr("Failed to create digital twin", err)
	}

	// Publish device registered event
	event := message.NewMessage(
		uuid.New().String(),
		"system",
		"*",
		message.TypeEvent,
		[]byte(`{"event":"device_registered","device_id":"`+dev.ID+`"}`),
	)
	event.SetContentType("application/json")

	if err := s.messageBroker.Publish(ctx, "events.device.registered", event); err != nil {
		s.logger.Warn("Failed to publish device registered event", "error", err)
	}

	return dev.ID, nil
}

// GetDevice retrieves a device by ID
func (s *Service) GetDevice(ctx context.Context, id string) (*device.Device, error) {
	dev, err := s.deviceRepo.Get(ctx, id)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to get device", err)
	}
	return dev, nil
}

// UpdateDevice updates an existing device
func (s *Service) UpdateDevice(ctx context.Context, dev *device.Device) error {
	// Update timestamp
	dev.UpdatedAt = time.Now()

	if err := s.deviceRepo.Update(ctx, dev); err != nil {
		return errbuilder.GenericErr("Failed to update device", err)
	}

	// Publish device updated event
	event := message.NewMessage(
		uuid.New().String(),
		"system",
		"*",
		message.TypeEvent,
		[]byte(`{"event":"device_updated","device_id":"`+dev.ID+`"}`),
	)
	event.SetContentType("application/json")

	if err := s.messageBroker.Publish(ctx, "events.device.updated", event); err != nil {
		s.logger.Warn("Failed to publish device updated event", "error", err)
	}

	return nil
}

// DeleteDevice deletes a device and its associated resources
func (s *Service) DeleteDevice(ctx context.Context, id string) error {
	// Delete digital twin
	if err := s.twinRepo.Delete(ctx, id); err != nil {
		s.logger.Warn("Failed to delete digital twin", "device_id", id, "error", err)
	}

	// Delete device
	if err := s.deviceRepo.Delete(ctx, id); err != nil {
		return errbuilder.GenericErr("Failed to delete device", err)
	}

	// Publish device deleted event
	event := message.NewMessage(
		uuid.New().String(),
		"system",
		"*",
		message.TypeEvent,
		[]byte(`{"event":"device_deleted","device_id":"`+id+`"}`),
	)
	event.SetContentType("application/json")

	if err := s.messageBroker.Publish(ctx, "events.device.deleted", event); err != nil {
		s.logger.Warn("Failed to publish device deleted event", "error", err)
	}

	return nil
}

// ListDevices lists devices with optional filtering
func (s *Service) ListDevices(ctx context.Context, filter repositories.DeviceFilter) ([]*device.Device, error) {
	devices, err := s.deviceRepo.List(ctx, filter)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to list devices", err)
	}
	return devices, nil
}

// GetDeviceTwin retrieves the digital twin for a device
func (s *Service) GetDeviceTwin(ctx context.Context, deviceID string) (*twin.DigitalTwin, error) {
	dt, err := s.twinRepo.Get(ctx, deviceID)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to get device twin", err)
	}
	return dt, nil
}

// UpdateDeviceState updates the state of a device's digital twin
func (s *Service) UpdateDeviceState(ctx context.Context, deviceID string, state map[string]any) error {
	if err := s.twinRepo.UpdateState(ctx, deviceID, state); err != nil {
		return errbuilder.GenericErr("Failed to update device state", err)
	}

	// Update device last seen timestamp
	dev, err := s.deviceRepo.Get(ctx, deviceID)
	if err == nil {
		dev.UpdateLastSeen()
		if err := s.deviceRepo.Update(ctx, dev); err != nil {
			s.logger.Warn("Failed to update device last seen timestamp", "device_id", deviceID, "error", err)
		}
	}

	return nil
}

// UpdateDesiredState updates the desired state of a device's digital twin
func (s *Service) UpdateDesiredState(ctx context.Context, deviceID string, desired map[string]any) error {
	if err := s.twinRepo.UpdateDesired(ctx, deviceID, desired); err != nil {
		return errbuilder.GenericErr("Failed to update desired state", err)
	}

	// Get the updated twin to check for delta
	dt, err := s.twinRepo.Get(ctx, deviceID)
	if err != nil {
		return errbuilder.GenericErr("Failed to get device twin after update", err)
	}

	// If there's a delta between desired and current state, send a command to the device
	delta := dt.GetStateDelta()
	if len(delta) > 0 {
		// TODO: Implement command sending to device
	}

	return nil
}
