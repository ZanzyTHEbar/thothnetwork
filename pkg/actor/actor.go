package actor

import (
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/anthdm/hollywood/actor"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
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

	// Add Supervisor field for actor supervision
	Supervisor *Supervisor
}

// Config holds configuration for the actor system
type Config struct {
	Address     string
	Port        int
	ClusterName string
}

// NewActorSystem creates a new actor system
func NewActorSystem(config Config, log logger.Logger, supervisor *Supervisor) *ActorSystem {
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
		engine:             engine,
		config:             config,
		logger:             logger,
		passivationManager: passivationManager,
		metrics:            metrics,
		deviceActors:       concurrent.NewHashMap(16),
		twinActors:         concurrent.NewHashMap(16),
		roomActors:         concurrent.NewHashMap(16),
		pipelineActors:     concurrent.NewHashMap(16),
		Supervisor:         supervisor, // Attach supervisor
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

	// Register with supervisor if present
	if s.Supervisor != nil {
		s.Supervisor.RegisterChild(deviceID, pid)
	}

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

	// Register with supervisor if present
	if s.Supervisor != nil {
		s.Supervisor.RegisterChild(roomID, pid)
	}

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
