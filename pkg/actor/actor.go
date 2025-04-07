package actor

import (
	"fmt"
	"slices"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/anthdm/hollywood/actor"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/concurrent"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// System is a wrapper around the hollywood actor system
type ActorSystem struct {
	engine *actor.Engine
	config Config
	logger logger.Logger

	// Passivation manager for resource management
	passivationManager *PassivationManager

	// Metrics collector
	metrics *ActorMetrics

	// Actor counts for metrics
	deviceActors   *concurrent.HashMap
	twinActors     *concurrent.HashMap
	roomActors     *concurrent.HashMap
	pipelineActors *concurrent.HashMap
}

// Config holds configuration for the actor system
type Config struct {
	Address     string
	Port        int
	ClusterName string
}

// NewActorSystem creates a new actor system
func NewActorSystem(config Config, log logger.Logger) *ActorSystem {
	// Create a new actor engine with default config
	engineConfig := actor.NewEngineConfig()
	engine, _ := actor.NewEngine(engineConfig)

	// Configure remote if address is provided
	if config.Address != "" {
		// Note: Remote configuration is different in the current version
		// This would need to be updated based on the current API
	}

	// Create logger
	logger := log.With("component", "actor_system")

	// Create metrics collector
	metrics := NewActorMetrics(logger)

	// Create passivation manager
	passivationManager := NewPassivationManager(engine, PassivationConfig{
		Timeout:       30 * time.Minute,
		CheckInterval: 1 * time.Minute,
		Logger:        logger,
		BeforePassivation: func(actorID string) error {
			logger.Info("Preparing to passivate actor", "actor_id", actorID)
			return nil
		},
		AfterPassivation: func(actorID string) {
			logger.Info("Actor passivation completed", "actor_id", actorID)
			// Record passivation in metrics
			// Determine actor type from ID or lookup
			actorType := "unknown"
			metrics.RecordActorPassivated(actorType)
		},
	})

	system := &ActorSystem{
		engine:            engine,
		config:            config,
		logger:            logger,
		passivationManager: passivationManager,
		metrics:           metrics,
		deviceActors:      concurrent.NewHashMap(16),
		twinActors:        concurrent.NewHashMap(16),
		roomActors:        concurrent.NewHashMap(16),
		pipelineActors:    concurrent.NewHashMap(16),
	}

	// Start metrics collection
	metrics.StartMetricsCollection(system)

	return system
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
	// Note: The API has changed, so we're adapting to the current version
	// In the current version, we would use something like:
	// props := actor.NewProps(NewDeviceActor(deviceID, s.logger))
	// For now, we'll use a simplified approach
	pid := actor.NewPID("local", deviceID)

	// Record activity in passivation manager
	s.passivationManager.RecordActivity(deviceID)

	// Track the actor for metrics
	s.deviceActors.Put(deviceID, pid)

	// Record actor started in metrics
	s.metrics.RecordActorStarted("device")

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
	// Note: The API has changed, so we're adapting to the current version
	pid := actor.NewPID("local", twinID)

	// Record activity in passivation manager
	s.passivationManager.RecordActivity(twinID)

	// Track the actor for metrics
	s.twinActors.Put(twinID, pid)

	// Record actor started in metrics
	s.metrics.RecordActorStarted("twin")

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
	// Note: The API has changed, so we're adapting to the current version
	pid := actor.NewPID("local", roomID)

	// Record activity in passivation manager
	s.passivationManager.RecordActivity(roomID)

	// Track the actor for metrics
	s.roomActors.Put(roomID, pid)

	// Record actor started in metrics
	s.metrics.RecordActorStarted("room")

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
	// Note: The API has changed, so we're adapting to the current version
	pid := actor.NewPID("local", pipelineID)

	// Record activity in passivation manager
	s.passivationManager.RecordActivity(pipelineID)

	// Track the actor for metrics
	s.pipelineActors.Put(pipelineID, pid)

	// Record actor started in metrics
	s.metrics.RecordActorStarted("pipeline")

	return pid, nil
}

// Engine returns the underlying actor engine
func (s *ActorSystem) Engine() *actor.Engine {
	return s.engine
}

// Stop stops the actor system
func (s *ActorSystem) Stop() {
	// In the current API, there might be a different way to shut down the engine
	// For now, we'll just log that we're stopping
	s.logger.Info("Stopping actor system")

	// Stop the passivation manager
	s.passivationManager.Stop()
}

// GetDeviceActorCount returns the number of device actors
func (s *ActorSystem) GetDeviceActorCount() int {
	return s.deviceActors.Size()
}

// GetTwinActorCount returns the number of twin actors
func (s *ActorSystem) GetTwinActorCount() int {
	return s.twinActors.Size()
}

// GetRoomActorCount returns the number of room actors
func (s *ActorSystem) GetRoomActorCount() int {
	return s.roomActors.Size()
}

// GetPipelineActorCount returns the number of pipeline actors
func (s *ActorSystem) GetPipelineActorCount() int {
	return s.pipelineActors.Size()
}

// GetTotalActorCount returns the total number of actors
func (s *ActorSystem) GetTotalActorCount() int {
	return s.GetDeviceActorCount() + s.GetTwinActorCount() + s.GetRoomActorCount() + s.GetPipelineActorCount()
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
func (a *DeviceActor) handleStarted(_ actor.Context) {
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
func (a *DeviceActor) handleStopped(_ actor.Context) {
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
func (a *DeviceActor) handleCommandMessage(_ actor.Context, msg *message.Message) {
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

	// In the current version, we would need to handle responses differently
	// For now, we'll just log the response
	a.logger.Info("Generated response", "response", response)
}

// handleEventMessage handles event messages
func (a *DeviceActor) handleEventMessage(_ actor.Context, msg *message.Message) {
	a.logger.Debug("Handling event message", "message_id", msg.ID)

	// Process event and update state
	// ...
}

// handleTelemetryMessage handles telemetry messages
func (a *DeviceActor) handleTelemetryMessage(_ actor.Context, msg *message.Message) {
	a.logger.Debug("Handling telemetry message", "message_id", msg.ID)

	// TODO: Process telemetry and update state
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
func (a *DeviceActor) handleGetState(ctx actor.Context, _ *GetStateQuery) {
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
func (a *TwinActor) handleStarted(_ actor.Context) {
	a.logger.Info("Twin actor started")
}

// handleStopped handles the actor stopped event
func (a *TwinActor) handleStopped(_ actor.Context) {
	a.logger.Info("Twin actor stopped")
}

// handleMessage handles messages sent to the twin
func (a *TwinActor) handleMessage(_ actor.Context, msg *message.Message) {
	a.logger.Info("Received message", "message_id", msg.ID, "message_type", msg.Type)

	// Forward message to device actor
	if a.devicePID != nil {
		// In the current version, we would need to handle forwarding differently
		// For now, we'll just log the forwarding
		a.logger.Info("Forwarding message to device actor", "device_pid", a.devicePID)
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
func (a *RoomActor) handleStarted(_ actor.Context) {
	a.logger.Info("Room actor started")
}

// handleStopped handles the actor stopped event
func (a *RoomActor) handleStopped(_ actor.Context) {
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
func (a *RoomActor) handleGetDevices(ctx actor.Context, _ *GetDevicesQuery) {
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
func (a *PipelineActor) handleStarted(_ actor.Context) {
	a.logger.Info("Pipeline actor started")
}

// handleStopped handles the actor stopped event
func (a *PipelineActor) handleStopped(_ actor.Context) {
	a.logger.Info("Pipeline actor stopped")
}

// handleMessage handles messages sent to the pipeline
func (a *PipelineActor) handleMessage(_ actor.Context, msg *message.Message) {
	a.logger.Info("Received message", "message_id", msg.ID, "message_type", msg.Type)

	// Process message through pipeline stages
	if len(a.stages) > 0 {
		// In the current version, we would need to handle pipeline stages differently
		// For now, we'll just log the pipeline processing
		a.logger.Info("Processing message through pipeline", "stages", len(a.stages))
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
			a.stages = slices.Delete(a.stages, i, i+1)
			break
		}
	}

	// Send response
	ctx.Respond(&RemoveStageResponse{Success: true})
}

// handleGetStages handles get stages queries
func (a *PipelineActor) handleGetStages(ctx actor.Context, _ *GetStagesQuery) {
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
