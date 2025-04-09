package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/adapters"
	thothnetworkv1 "github.com/ZanzyTHEbar/thothnetwork/pkg/gen/proto/thothnetwork/v1"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Config holds configuration for the gRPC adapter
type Config struct {
	Host string
	Port int
}

// Adapter is a gRPC implementation of the ProtocolAdapter interface
type Adapter struct {
	config          Config
	server          *grpc.Server
	handler         adapters.MessageHandler
	logger          logger.Logger
	status          adapters.AdapterStatus
	mu              sync.RWMutex
	deviceService   *DeviceService
	messageService  *MessageService
	roomService     *RoomService
	twinService     *TwinService
	pipelineService *PipelineService
}

// NewAdapter creates a new gRPC adapter
func NewAdapter(config Config, logger logger.Logger) *Adapter {
	return &Adapter{
		config: config,
		logger: logger.With("component", "grpc_adapter"),
		status: adapters.StatusStopped,
	}
}

// Start starts the adapter
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if already started
	if a.status != adapters.StatusStopped {
		return errbuilder.GenericErr("Adapter already started", nil)
	}

	a.status = adapters.StatusStarting

	// Create the listener
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		a.status = adapters.StatusError
		return errbuilder.GenericErr("Failed to create gRPC listener", err)
	}

	// Create gRPC server
	a.server = grpc.NewServer()

	// Create and register service implementations
	a.deviceService = NewDeviceService(a)
	a.messageService = NewMessageService(a)
	a.roomService = NewRoomService(a)
	a.twinService = NewTwinService(a)
	a.pipelineService = NewPipelineService(a)

	// Register the service implementations with the gRPC server
	thothnetworkv1.RegisterDeviceServiceServer(a.server, a.deviceService)
	thothnetworkv1.RegisterMessageServiceServer(a.server, a.messageService)
	thothnetworkv1.RegisterRoomServiceServer(a.server, a.roomService)
	thothnetworkv1.RegisterTwinServiceServer(a.server, a.twinService)
	thothnetworkv1.RegisterPipelineServiceServer(a.server, a.pipelineService)

	// Register reflection service for gRPC CLI and other tools
	reflection.Register(a.server)

	// Start server in a goroutine
	go func() {
		a.logger.Info("Starting gRPC server", "addr", addr)

		if err := a.server.Serve(lis); err != nil {
			a.logger.Error("gRPC server error", "error", err)
			a.mu.Lock()
			a.status = adapters.StatusError
			a.mu.Unlock()
		}
	}()

	a.status = adapters.AdapterStatus(adapters.StatusRunning)
	a.logger.Info("gRPC adapter started")

	return nil
}

// Stop stops the adapter
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status == adapters.StatusStopped {
		return nil
	}

	if a.server != nil {
		a.server.GracefulStop()
	}

	a.status = adapters.StatusStopped
	a.logger.Info("gRPC adapter stopped")

	return nil
}

// Status returns the current status of the adapter
func (a *Adapter) Status() adapters.AdapterStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// SetMessageHandler sets the handler for incoming messages
func (a *Adapter) SetMessageHandler(handler adapters.MessageHandler) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.status != adapters.StatusStopped {
		return errbuilder.GenericErr("Cannot set message handler while adapter is running", nil)
	}

	a.handler = handler
	return nil
}

// SendMessage sends a message to a device
func (a *Adapter) SendMessage(ctx context.Context, deviceID string, msg *message.Message) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.status != adapters.StatusRunning {
		return errbuilder.GenericErr("Adapter not running", nil)
	}

	// In a gRPC adapter, this would likely be implemented differently
	// For now, log that we received the request
	a.logger.Debug("Received request to send message",
		"deviceID", deviceID,
		"messageID", msg.ID,
		"source", msg.Source,
		"target", msg.Target,
		"type", msg.Type)

	// The actual implementation would depend on how clients connect to the gRPC server
	// and how you manage device connections
	return nil
}
