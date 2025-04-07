package adapter

import (
	"context"
	"sync"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/adapters"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/brokers"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Service manages protocol adapters
type Service struct {
	adapters map[string]adapters.ProtocolAdapter
	broker   brokers.MessageBroker
	logger   logger.Logger
	mu       sync.RWMutex
}

// NewService creates a new adapter service
func NewService(broker brokers.MessageBroker, logger logger.Logger) *Service {
	return &Service{
		adapters: make(map[string]adapters.ProtocolAdapter),
		broker:   broker,
		logger:   logger.With("component", "adapter_service"),
	}
}

// RegisterAdapter registers a protocol adapter
func (s *Service) RegisterAdapter(name string, adapter adapters.ProtocolAdapter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if adapter already registered
	if _, exists := s.adapters[name]; exists {
		return errbuilder.GenericErr("Adapter already registered", nil)
	}

	// Set message handler
	if err := adapter.SetMessageHandler(s.handleMessage); err != nil {
		return errbuilder.GenericErr("Failed to set message handler", err)
	}

	// Register adapter
	s.adapters[name] = adapter
	s.logger.Info("Adapter registered", "name", name)

	return nil
}

// StartAdapter starts a protocol adapter
func (s *Service) StartAdapter(ctx context.Context, name string) error {
	s.mu.RLock()
	adapter, exists := s.adapters[name]
	s.mu.RUnlock()

	if !exists {
		return errbuilder.GenericErr("Adapter not found", nil)
	}

	// Start adapter
	if err := adapter.Start(ctx); err != nil {
		return errbuilder.GenericErr("Failed to start adapter", err)
	}

	s.logger.Info("Adapter started", "name", name)
	return nil
}

// StopAdapter stops a protocol adapter
func (s *Service) StopAdapter(ctx context.Context, name string) error {
	s.mu.RLock()
	adapter, exists := s.adapters[name]
	s.mu.RUnlock()

	if !exists {
		return errbuilder.GenericErr("Adapter not found", nil)
	}

	// Stop adapter
	if err := adapter.Stop(ctx); err != nil {
		return errbuilder.GenericErr("Failed to stop adapter", err)
	}

	s.logger.Info("Adapter stopped", "name", name)
	return nil
}

// GetAdapterStatus returns the status of a protocol adapter
func (s *Service) GetAdapterStatus(name string) (adapters.AdapterStatus, error) {
	s.mu.RLock()
	adapter, exists := s.adapters[name]
	s.mu.RUnlock()

	if !exists {
		return "", errbuilder.GenericErr("Adapter not found", nil)
	}

	return adapter.Status(), nil
}

// ListAdapters returns a list of registered adapters and their statuses
func (s *Service) ListAdapters() map[string]adapters.AdapterStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]adapters.AdapterStatus)
	for name, adapter := range s.adapters {
		result[name] = adapter.Status()
	}

	return result
}

// SendMessageToDevice sends a message to a device through the appropriate adapter
func (s *Service) SendMessageToDevice(ctx context.Context, deviceID string, msg *message.Message) error {
	// TODO: In a real implementation, we would determine which adapter to use based on the device ID
	// For now, we'll just publish the message to the broker
	return s.broker.Publish(ctx, "devices."+deviceID, msg)
}

// handleMessage handles messages from protocol adapters
func (s *Service) handleMessage(ctx context.Context, msg *message.Message) error {
	// Check if this is a response message
	if msg.Type == message.TypeResponse {
		// Check if we have a response channel in the context
		if respChan, ok := ctx.Value("response_channel").(chan *message.Message); ok {
			// Send response to channel
			select {
			case respChan <- msg:
				return nil
			default:
				s.logger.Warn("Response channel full or closed")
			}
		}
	}

	// Determine topic based on message type and target
	var topic string
	switch msg.Type {
	case message.TypeTelemetry:
		topic = "telemetry." + msg.Source
	case message.TypeEvent:
		topic = "events." + msg.Source
	case message.TypeCommand:
		if msg.Target == "*" {
			topic = "commands.broadcast"
		} else {
			topic = "commands." + msg.Target
		}
	case message.TypeResponse:
		topic = "responses." + msg.Target
	default:
		return errbuilder.GenericErr("Unknown message type", nil)
	}

	// Publish message to broker
	return s.broker.Publish(ctx, topic, msg)
}
