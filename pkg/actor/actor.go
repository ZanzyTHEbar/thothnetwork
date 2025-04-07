package actor

import (
	"context"
	"fmt"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// ActorSystem is a wrapper around the hollywood actor system
type ActorSystem struct {
	engine *actor.Engine
	config Config
	logger logger.Logger
}

// Config holds configuration for the actor system
type Config struct {
	Address     string
	Port        int
	ClusterName string
}

// NewActorSystem creates a new actor system
func NewActorSystem(config Config, log logger.Logger) *ActorSystem {
	// Create a new actor engine
	engine := actor.NewEngine()

	// Configure remote if address is provided
	if config.Address != "" {
		remoteConfig := remote.Config{
			DefaultSerializerID: 0, // Use the default serializer
		}

		// Create a new remote engine
		remote.NewEngine(engine, remote.NewNetworkTransport(fmt.Sprintf("%s:%d", config.Address, config.Port)), remoteConfig)
	}

	return &ActorSystem{
		engine: engine,
		config: config,
		logger: log.With("component", "actor_system"),
	}
}

// SpawnDevice spawns a new device actor
func (s *ActorSystem) SpawnDevice(deviceID string) (*actor.PID, error) {
	// Check if device ID is provided
	if deviceID == "" {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Device ID is required").
			WithCode(errbuilder.CodeInvalidArgument)
	}

	// Create a new device actor
	props := actor.PropsFromProducer(func() actor.Receiver {
		return NewDeviceActor(deviceID, s.logger)
	})

	// Spawn the actor
	pid := s.engine.Spawn(props, deviceID)

	return pid, nil
}

// SpawnTwin spawns a new digital twin actor
func (s *ActorSystem) SpawnTwin(twinID string, devicePID *actor.PID) (*actor.PID, error) {
	// Check if twin ID is provided
	if twinID == "" {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Twin ID is required").
			WithCode(errbuilder.CodeInvalidArgument)
	}

	// Create a new twin actor
	props := actor.PropsFromProducer(func() actor.Receiver {
		return NewTwinActor(twinID, devicePID, s.logger)
	})

	// Spawn the actor
	pid := s.engine.Spawn(props, twinID)

	return pid, nil
}

// SpawnRoom spawns a new room actor
func (s *ActorSystem) SpawnRoom(roomID string) (*actor.PID, error) {
	// Check if room ID is provided
	if roomID == "" {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Room ID is required").
			WithCode(errbuilder.CodeInvalidArgument)
	}

	// Create a new room actor
	props := actor.PropsFromProducer(func() actor.Receiver {
		return NewRoomActor(roomID, s.logger)
	})

	// Spawn the actor
	pid := s.engine.Spawn(props, roomID)

	return pid, nil
}

// SpawnPipeline spawns a new pipeline actor
func (s *ActorSystem) SpawnPipeline(pipelineID string) (*actor.PID, error) {
	// Check if pipeline ID is provided
	if pipelineID == "" {
		return nil, errbuilder.NewErrBuilder().
			WithMsg("Pipeline ID is required").
			WithCode(errbuilder.CodeInvalidArgument)
	}

	// Create a new pipeline actor
	props := actor.PropsFromProducer(func() actor.Receiver {
		return NewPipelineActor(pipelineID, s.logger)
	})

	// Spawn the actor
	pid := s.engine.Spawn(props, pipelineID)

	return pid, nil
}

// Stop stops the actor system
func (s *ActorSystem) Stop() {
	s.engine.Shutdown()
}

// DeviceActor represents a device in the system
type DeviceActor struct {
	deviceID string
	state    *device.Device
	logger   logger.Logger
}

// NewDeviceActor creates a new device actor
func NewDeviceActor(deviceID string, log logger.Logger) *DeviceActor {
	return &DeviceActor{
		deviceID: deviceID,
		logger:   log.With("actor", "device", "device_id", deviceID),
	}
}

// Receive handles messages sent to the device actor
func (a *DeviceActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.handleStarted(ctx)
	case *actor.Stopped:
		a.handleStopped(ctx)
	case *message.Message:
		a.handleMessage(ctx, msg)
	case *UpdateStateCommand:
		a.handleUpdateState(ctx, msg)
	case *GetStateQuery:
		a.handleGetState(ctx, msg)
	default:
		a.logger.Warn("Received unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

// handleStarted handles the actor started event
func (a *DeviceActor) handleStarted(ctx actor.Context) {
	a.logger.Info("Device actor started")

	// Initialize state if not already initialized
	if a.state == nil {
		a.state = &device.Device{
			ID:        a.deviceID,
			Status:    device.StatusOffline,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]string),
		}
	}
}

// handleStopped handles the actor stopped event
func (a *DeviceActor) handleStopped(ctx actor.Context) {
	a.logger.Info("Device actor stopped")
}

// handleMessage handles messages sent to the device
func (a *DeviceActor) handleMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Info("Received message", "message_id", msg.ID, "message_type", msg.Type)

	// Process message based on type
	switch msg.Type {
	case "command":
		// Handle command message
		a.handleCommandMessage(ctx, msg)
	case "event":
		// Handle event message
		a.handleEventMessage(ctx, msg)
	case "telemetry":
		// Handle telemetry message
		a.handleTelemetryMessage(ctx, msg)
	default:
		a.logger.Warn("Unknown message type", "type", msg.Type)
	}
}

// handleCommandMessage handles command messages
func (a *DeviceActor) handleCommandMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling command message", "message_id", msg.ID)

	// Process command and update state
	// ...

	// Send response
	response := &message.Message{
		ID:      uuid.New().String(),
		Type:    "response",
		Source:  a.deviceID,
		Target:  msg.Source,
		Payload: []byte(`{"status": "success"}`),
	}

	// Send response to sender
	if msg.ReplyTo != nil {
		ctx.Send(msg.ReplyTo, response)
	}
}

// handleEventMessage handles event messages
func (a *DeviceActor) handleEventMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling event message", "message_id", msg.ID)

	// Process event and update state
	// ...
}

// handleTelemetryMessage handles telemetry messages
func (a *DeviceActor) handleTelemetryMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling telemetry message", "message_id", msg.ID)

	// Process telemetry and update state
	// ...
}

// handleUpdateState handles state update commands
func (a *DeviceActor) handleUpdateState(ctx actor.Context, cmd *UpdateStateCommand) {
	a.logger.Debug("Updating device state")

	// Update state
	a.state = cmd.State
	a.state.UpdatedAt = time.Now()

	// Send response
	ctx.Respond(&UpdateStateResponse{Success: true})
}

// handleGetState handles state queries
func (a *DeviceActor) handleGetState(ctx actor.Context, query *GetStateQuery) {
	a.logger.Debug("Getting device state")

	// Send current state
	ctx.Respond(&GetStateResponse{State: a.state})
}

// TwinActor represents a digital twin in the system
type TwinActor struct {
	twinID    string
	devicePID *actor.PID
	logger    logger.Logger
}

// NewTwinActor creates a new twin actor
func NewTwinActor(twinID string, devicePID *actor.PID, log logger.Logger) *TwinActor {
	return &TwinActor{
		twinID:    twinID,
		devicePID: devicePID,
		logger:    log.With("actor", "twin", "twin_id", twinID),
	}
}

// Receive handles messages sent to the twin actor
func (a *TwinActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.handleStarted(ctx)
	case *actor.Stopped:
		a.handleStopped(ctx)
	case *message.Message:
		a.handleMessage(ctx, msg)
	default:
		a.logger.Warn("Received unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

// handleStarted handles the actor started event
func (a *TwinActor) handleStarted(ctx actor.Context) {
	a.logger.Info("Twin actor started")
}

// handleStopped handles the actor stopped event
func (a *TwinActor) handleStopped(ctx actor.Context) {
	a.logger.Info("Twin actor stopped")
}

// handleMessage handles messages sent to the twin
func (a *TwinActor) handleMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Info("Received message", "message_id", msg.ID, "message_type", msg.Type)

	// Forward message to device actor
	if a.devicePID != nil {
		ctx.Request(a.devicePID, msg, ctx.Self())
	}
}

// RoomActor represents a room in the system
type RoomActor struct {
	roomID  string
	devices map[string]*actor.PID
	logger  logger.Logger
}

// NewRoomActor creates a new room actor
func NewRoomActor(roomID string, log logger.Logger) *RoomActor {
	return &RoomActor{
		roomID:  roomID,
		devices: make(map[string]*actor.PID),
		logger:  log.With("actor", "room", "room_id", roomID),
	}
}

// Receive handles messages sent to the room actor
func (a *RoomActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.handleStarted(ctx)
	case *actor.Stopped:
		a.handleStopped(ctx)
	case *message.Message:
		a.handleMessage(ctx, msg)
	case *AddDeviceCommand:
		a.handleAddDevice(ctx, msg)
	case *RemoveDeviceCommand:
		a.handleRemoveDevice(ctx, msg)
	case *GetDevicesQuery:
		a.handleGetDevices(ctx, msg)
	default:
		a.logger.Warn("Received unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

// handleStarted handles the actor started event
func (a *RoomActor) handleStarted(ctx actor.Context) {
	a.logger.Info("Room actor started")
}

// handleStopped handles the actor stopped event
func (a *RoomActor) handleStopped(ctx actor.Context) {
	a.logger.Info("Room actor stopped")
}

// handleMessage handles messages sent to the room
func (a *RoomActor) handleMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Info("Received message", "message_id", msg.ID, "message_type", msg.Type)

	// Broadcast message to all devices in the room
	for deviceID, devicePID := range a.devices {
		a.logger.Debug("Forwarding message to device", "device_id", deviceID)
		ctx.Send(devicePID, msg)
	}
}

// handleAddDevice handles add device commands
func (a *RoomActor) handleAddDevice(ctx actor.Context, cmd *AddDeviceCommand) {
	a.logger.Debug("Adding device to room", "device_id", cmd.DeviceID)

	// Add device to room
	a.devices[cmd.DeviceID] = cmd.DevicePID

	// Send response
	ctx.Respond(&AddDeviceResponse{Success: true})
}

// handleRemoveDevice handles remove device commands
func (a *RoomActor) handleRemoveDevice(ctx actor.Context, cmd *RemoveDeviceCommand) {
	a.logger.Debug("Removing device from room", "device_id", cmd.DeviceID)

	// Remove device from room
	delete(a.devices, cmd.DeviceID)

	// Send response
	ctx.Respond(&RemoveDeviceResponse{Success: true})
}

// handleGetDevices handles get devices queries
func (a *RoomActor) handleGetDevices(ctx actor.Context, query *GetDevicesQuery) {
	a.logger.Debug("Getting devices in room")

	// Get devices
	deviceIDs := make([]string, 0, len(a.devices))
	for deviceID := range a.devices {
		deviceIDs = append(deviceIDs, deviceID)
	}

	// Send response
	ctx.Respond(&GetDevicesResponse{DeviceIDs: deviceIDs})
}

// PipelineActor represents a processing pipeline in the system
type PipelineActor struct {
	pipelineID string
	stages     []*actor.PID
	logger     logger.Logger
}

// NewPipelineActor creates a new pipeline actor
func NewPipelineActor(pipelineID string, log logger.Logger) *PipelineActor {
	return &PipelineActor{
		pipelineID: pipelineID,
		stages:     make([]*actor.PID, 0),
		logger:     log.With("actor", "pipeline", "pipeline_id", pipelineID),
	}
}

// Receive handles messages sent to the pipeline actor
func (a *PipelineActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.handleStarted(ctx)
	case *actor.Stopped:
		a.handleStopped(ctx)
	case *message.Message:
		a.handleMessage(ctx, msg)
	case *AddStageCommand:
		a.handleAddStage(ctx, msg)
	case *RemoveStageCommand:
		a.handleRemoveStage(ctx, msg)
	case *GetStagesQuery:
		a.handleGetStages(ctx, msg)
	default:
		a.logger.Warn("Received unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

// handleStarted handles the actor started event
func (a *PipelineActor) handleStarted(ctx actor.Context) {
	a.logger.Info("Pipeline actor started")
}

// handleStopped handles the actor stopped event
func (a *PipelineActor) handleStopped(ctx actor.Context) {
	a.logger.Info("Pipeline actor stopped")
}

// handleMessage handles messages sent to the pipeline
func (a *PipelineActor) handleMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Info("Received message", "message_id", msg.ID, "message_type", msg.Type)

	// Process message through pipeline stages
	if len(a.stages) > 0 {
		// Send to first stage
		ctx.Request(a.stages[0], msg, ctx.Self())
	}
}

// handleAddStage handles add stage commands
func (a *PipelineActor) handleAddStage(ctx actor.Context, cmd *AddStageCommand) {
	a.logger.Debug("Adding stage to pipeline", "stage_id", cmd.StageID)

	// Add stage to pipeline
	a.stages = append(a.stages, cmd.StagePID)

	// Send response
	ctx.Respond(&AddStageResponse{Success: true})
}

// handleRemoveStage handles remove stage commands
func (a *PipelineActor) handleRemoveStage(ctx actor.Context, cmd *RemoveStageCommand) {
	a.logger.Debug("Removing stage from pipeline", "stage_id", cmd.StageID)

	// Find stage
	for i, stagePID := range a.stages {
		if stagePID.ID == cmd.StageID {
			// Remove stage
			a.stages = append(a.stages[:i], a.stages[i+1:]...)
			break
		}
	}

	// Send response
	ctx.Respond(&RemoveStageResponse{Success: true})
}

// handleGetStages handles get stages queries
func (a *PipelineActor) handleGetStages(ctx actor.Context, query *GetStagesQuery) {
	a.logger.Debug("Getting stages in pipeline")

	// Get stages
	stageIDs := make([]string, 0, len(a.stages))
	for _, stagePID := range a.stages {
		stageIDs = append(stageIDs, stagePID.ID)
	}

	// Send response
	ctx.Respond(&GetStagesResponse{StageIDs: stageIDs})
}

// Commands and Queries

// UpdateStateCommand is a command to update a device's state
type UpdateStateCommand struct {
	State *device.Device
}

// UpdateStateResponse is a response to an update state command
type UpdateStateResponse struct {
	Success bool
}

// GetStateQuery is a query to get a device's state
type GetStateQuery struct{}

// GetStateResponse is a response to a get state query
type GetStateResponse struct {
	State *device.Device
}

// AddDeviceCommand is a command to add a device to a room
type AddDeviceCommand struct {
	DeviceID  string
	DevicePID *actor.PID
}

// AddDeviceResponse is a response to an add device command
type AddDeviceResponse struct {
	Success bool
}

// RemoveDeviceCommand is a command to remove a device from a room
type RemoveDeviceCommand struct {
	DeviceID string
}

// RemoveDeviceResponse is a response to a remove device command
type RemoveDeviceResponse struct {
	Success bool
}

// GetDevicesQuery is a query to get devices in a room
type GetDevicesQuery struct{}

// GetDevicesResponse is a response to a get devices query
type GetDevicesResponse struct {
	DeviceIDs []string
}

// AddStageCommand is a command to add a stage to a pipeline
type AddStageCommand struct {
	StageID  string
	StagePID *actor.PID
}

// AddStageResponse is a response to an add stage command
type AddStageResponse struct {
	Success bool
}

// RemoveStageCommand is a command to remove a stage from a pipeline
type RemoveStageCommand struct {
	StageID string
}

// RemoveStageResponse is a response to a remove stage command
type RemoveStageResponse struct {
	Success bool
}

// GetStagesQuery is a query to get stages in a pipeline
type GetStagesQuery struct{}

// GetStagesResponse is a response to a get stages query
type GetStagesResponse struct {
	StageIDs []string
}
