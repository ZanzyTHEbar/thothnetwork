package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZanzyTHEbar/thothnetwork/internal/adapters/brokers/nats"
	httpAdapter "github.com/ZanzyTHEbar/thothnetwork/internal/adapters/protocols/http"
	mqttAdapter "github.com/ZanzyTHEbar/thothnetwork/internal/adapters/protocols/mqtt"
	wsAdapter "github.com/ZanzyTHEbar/thothnetwork/internal/adapters/protocols/websocket"
	"github.com/ZanzyTHEbar/thothnetwork/internal/adapters/repositories/memory"
	"github.com/ZanzyTHEbar/thothnetwork/internal/config"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/pipeline"
	actorService "github.com/ZanzyTHEbar/thothnetwork/internal/services/actor"
	adapterService "github.com/ZanzyTHEbar/thothnetwork/internal/services/adapter"
	deviceService "github.com/ZanzyTHEbar/thothnetwork/internal/services/device"
	pipelineService "github.com/ZanzyTHEbar/thothnetwork/internal/services/pipeline"
	roomService "github.com/ZanzyTHEbar/thothnetwork/internal/services/room"
	streamService "github.com/ZanzyTHEbar/thothnetwork/internal/services/stream"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/actor"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/metrics"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/tracing"
)

// Server represents the ThothNetwork server and all its components
type Server struct {
	// Configuration
	Config *config.Config

	// Core components
	Logger           logger.Logger
	MetricsCollector *metrics.Metrics
	Tracer           *tracing.Tracer

	// Repositories
	DeviceRepo *memory.DeviceRepository
	TwinRepo   *memory.TwinRepository
	RoomRepo   *memory.RoomRepository

	// Message broker
	Broker *nats.MessageBroker

	// Services
	DeviceService   *deviceService.Service
	PipelineService *pipelineService.Service
	RoomService     *roomService.Service
	AdapterService  *adapterService.Service
	ActorService    *actorService.Service
	StreamService   *streamService.Service

	// Adapters
	HTTPAdapter      *httpAdapter.Adapter
	MQTTAdapter      *mqttAdapter.Adapter
	WebSocketAdapter *wsAdapter.Adapter

	// Actor system
	ActorSystem *actor.ActorSystem

	// Pipeline
	DefaultPipelineID string

	// Shutdown signal channel
	sigCh chan os.Signal
}

// NewServer creates a new ThothNetwork server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Create a new server instance
	server := &Server{
		Config: cfg,
		sigCh:  make(chan os.Signal, 1),
	}

	// Create logger
	log, err := logger.NewLogger(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: "stdout",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	server.Logger = log

	// Create metrics collector
	server.MetricsCollector = metrics.NewMetrics(metrics.Config{
		Host: cfg.Metrics.Host,
		Port: cfg.Metrics.Port,
	}, log)

	// Create tracer
	tracer, err := tracing.NewTracer(tracing.Config{
		ServiceName:    cfg.Tracing.ServiceName,
		ServiceVersion: cfg.Tracing.ServiceVersion,
		Endpoint:       cfg.Tracing.Endpoint,
		Enabled:        cfg.Tracing.Enabled,
	}, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}
	server.Tracer = tracer

	// Create repositories
	server.DeviceRepo = memory.NewDeviceRepository()
	server.TwinRepo = memory.NewTwinRepository()
	server.RoomRepo = memory.NewRoomRepository()

	// Create message broker
	server.Broker = nats.NewMessageBroker(nats.Config{
		URL:           cfg.NATS.URL,
		Username:      cfg.NATS.Username,
		Password:      cfg.NATS.Password,
		Token:         cfg.NATS.Token,
		MaxReconnects: cfg.NATS.MaxReconnects,
		ReconnectWait: cfg.NATS.ReconnectWait,
		Timeout:       cfg.NATS.Timeout,
	}, log)

	// Create actor system
	server.ActorSystem = actor.NewActorSystem(actor.Config{
		Address:     cfg.Actor.Address,
		Port:        cfg.Actor.Port,
		ClusterName: cfg.Actor.ClusterName,
	}, log)

	// Create services
	server.DeviceService = deviceService.NewService(server.DeviceRepo, server.TwinRepo, server.Broker, log)
	server.PipelineService = pipelineService.NewService(server.Broker, log)
	server.RoomService = roomService.NewService(server.RoomRepo, server.DeviceRepo, server.Broker, log)
	server.AdapterService = adapterService.NewService(server.Broker, log)
	server.ActorService = actorService.NewService(server.ActorSystem, server.DeviceRepo, server.RoomRepo, log)
	// Create stream service with nil pipeline for now
	server.StreamService = streamService.NewService(server.Broker, nil, log)

	// Create protocol adapters
	server.HTTPAdapter = httpAdapter.NewAdapter(httpAdapter.Config{
		Host:            cfg.HTTP.Host,
		Port:            cfg.HTTP.Port,
		BasePath:        cfg.HTTP.BasePath,
		ReadTimeout:     cfg.HTTP.ReadTimeout,
		WriteTimeout:    cfg.HTTP.WriteTimeout,
		MaxHeaderBytes:  cfg.HTTP.MaxHeaderBytes,
		ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}, log)

	server.MQTTAdapter = mqttAdapter.NewAdapter(mqttAdapter.Config{
		BrokerURL:            cfg.MQTT.BrokerURL,
		ClientID:             cfg.MQTT.ClientID,
		Username:             cfg.MQTT.Username,
		Password:             cfg.MQTT.Password,
		CleanSession:         cfg.MQTT.CleanSession,
		QoS:                  cfg.MQTT.QoS,
		ConnectTimeout:       cfg.MQTT.ConnectTimeout,
		KeepAlive:            cfg.MQTT.KeepAlive,
		PingTimeout:          cfg.MQTT.PingTimeout,
		ConnectRetryDelay:    cfg.MQTT.ConnectRetryDelay,
		MaxReconnectAttempts: cfg.MQTT.MaxReconnectAttempts,
		TopicPrefix:          cfg.MQTT.TopicPrefix,
	}, log)

	server.WebSocketAdapter = wsAdapter.NewAdapter(wsAdapter.Config{
		Host:            cfg.WebSocket.Host,
		Port:            cfg.WebSocket.Port,
		Path:            cfg.WebSocket.Path,
		ReadBufferSize:  cfg.WebSocket.ReadBufferSize,
		WriteBufferSize: cfg.WebSocket.WriteBufferSize,
		PingInterval:    cfg.WebSocket.PingInterval,
		PongWait:        cfg.WebSocket.PongWait,
	}, log)

	// TODO: Create gRPC adapter

	return server, nil
}

// Start starts the ThothNetwork server
func (s *Server) Start(ctx context.Context) error {
	// Connect to message broker
	if err := s.Broker.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Start metrics server
	if err := s.MetricsCollector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	// Register adapters
	if err := s.AdapterService.RegisterAdapter("http", s.HTTPAdapter); err != nil {
		return fmt.Errorf("failed to register HTTP adapter: %w", err)
	}
	if err := s.AdapterService.RegisterAdapter("mqtt", s.MQTTAdapter); err != nil {
		return fmt.Errorf("failed to register MQTT adapter: %w", err)
	}
	if err := s.AdapterService.RegisterAdapter("websocket", s.WebSocketAdapter); err != nil {
		return fmt.Errorf("failed to register WebSocket adapter: %w", err)
	}
	

	// Start adapters
	if err := s.AdapterService.StartAdapter(ctx, "http"); err != nil {
		return fmt.Errorf("failed to start HTTP adapter: %w", err)
	}
	if err := s.AdapterService.StartAdapter(ctx, "mqtt"); err != nil {
		return fmt.Errorf("failed to start MQTT adapter: %w", err)
	}
	if err := s.AdapterService.StartAdapter(ctx, "websocket"); err != nil {
		return fmt.Errorf("failed to start WebSocket adapter: %w", err)
	}
	if err := s.AdapterService.StartAdapter(ctx, "grpc"); err != nil {
		return fmt.Errorf("failed to start gRPC adapter: %w", err)
	}

	// Start actor service
	if err := s.ActorService.StartAllDeviceActors(ctx); err != nil {
		s.Logger.Warn("Failed to start all device actors", "error", err)
	}
	if err := s.ActorService.StartAllRoomActors(ctx); err != nil {
		s.Logger.Warn("Failed to start all room actors", "error", err)
	}

	// Create default pipeline
	stages := []pipeline.ProcessingStage{
		s.PipelineService.CreateValidationStage("message-validation", func(ctx context.Context, msg *message.Message) error {
			// Validate message
			if msg.Source == "" {
				return errors.New("message source is required")
			}
			if msg.Target == "" {
				return errors.New("message target is required")
			}
			return nil
		}),
		s.PipelineService.CreateTransformationStage("message-transformation", func(ctx context.Context, msg *message.Message) (*message.Message, error) {
			// Transform message
			msg.SetMetadata("processed_at", time.Now().Format(time.RFC3339))
			return msg, nil
		}),
	}

	pipelineID, err := s.PipelineService.CreatePipeline(ctx, "default", "telemetry.>", "processed.telemetry", stages)
	if err != nil {
		return fmt.Errorf("failed to create default pipeline: %w", err)
	}
	s.DefaultPipelineID = pipelineID

	// Start default pipeline
	if err := s.PipelineService.StartPipeline(ctx, pipelineID); err != nil {
		return fmt.Errorf("failed to start default pipeline: %w", err)
	}

	// Setup signal handling
	signal.Notify(s.sigCh, syscall.SIGINT, syscall.SIGTERM)

	return nil
}

// WaitForSignal blocks until a termination signal is received
func (s *Server) WaitForSignal() os.Signal {
	return <-s.sigCh
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.Logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	// Stop actor service
	s.ActorService.StopAllActors()

	// Stop adapters
	if err := s.AdapterService.StopAdapter(shutdownCtx, "grpc"); err != nil {
		s.Logger.Error("Failed to stop gRPC adapter", "error", err)
	}
	if err := s.AdapterService.StopAdapter(shutdownCtx, "websocket"); err != nil {
		s.Logger.Error("Failed to stop WebSocket adapter", "error", err)
	}
	if err := s.AdapterService.StopAdapter(shutdownCtx, "mqtt"); err != nil {
		s.Logger.Error("Failed to stop MQTT adapter", "error", err)
	}
	if err := s.AdapterService.StopAdapter(shutdownCtx, "http"); err != nil {
		s.Logger.Error("Failed to stop HTTP adapter", "error", err)
	}

	// Stop pipeline
	if s.DefaultPipelineID != "" {
		if err := s.PipelineService.StopPipeline(shutdownCtx, s.DefaultPipelineID); err != nil {
			s.Logger.Error("Failed to stop default pipeline", "error", err)
		}
	}

	// Disconnect from message broker
	if err := s.Broker.Disconnect(shutdownCtx); err != nil {
		s.Logger.Error("Failed to disconnect from NATS", "error", err)
	}

	// Stop metrics server
	if err := s.MetricsCollector.Stop(shutdownCtx); err != nil {
		s.Logger.Error("Failed to stop metrics server", "error", err)
	}

	// Shutdown tracer
	if err := s.Tracer.Shutdown(shutdownCtx); err != nil {
		s.Logger.Error("Failed to shutdown tracer", "error", err)
	}

	s.Logger.Info("Shutdown complete")
	return nil
}
