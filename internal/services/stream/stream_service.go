package stream

import (
	"context"
	"sync"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/brokers"
	"github.com/ZanzyTHEbar/thothnetwork/internal/services/pipeline"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// StreamConfig defines configuration for a stream
type StreamConfig struct {
	// Subjects is the list of subjects to include in the stream
	Subjects []string

	// ProcessingEnabled indicates whether messages should be processed through the pipeline
	ProcessingEnabled bool

	// RetentionPolicy defines how messages are retained
	RetentionPolicy string

	// MaxAge is the maximum age of messages in the stream (in seconds)
	MaxAge int64

	// MaxBytes is the maximum size of the stream (in bytes)
	MaxBytes int64

	// MaxMsgs is the maximum number of messages in the stream
	MaxMsgs int64
}

// Service provides stream management functionality
type Service struct {
	broker        brokers.MessageBroker
	pipeline      *pipeline.Pipeline
	logger        logger.Logger
	streams       map[string]StreamConfig
	subscriptions map[string]brokers.Subscription
	mu            sync.RWMutex
}

// NewService creates a new stream service
func NewService(
	broker brokers.MessageBroker,
	pipeline *pipeline.Pipeline,
	logger logger.Logger,
) *Service {
	return &Service{
		broker:        broker,
		pipeline:      pipeline,
		logger:        logger.With("service", "stream"),
		streams:       make(map[string]StreamConfig),
		subscriptions: make(map[string]brokers.Subscription),
	}
}

// CreateStream creates a new stream
func (s *Service) CreateStream(name string, subjects []string) error {
	s.logger.Info("Creating stream", "name", name, "subjects", subjects)

	// Validate inputs
	if name == "" {
		return errbuilder.NewErrBuilder().
			WithMsg("Stream name is required").
			WithCode(errbuilder.CodeInvalidArgument)
	}

	if len(subjects) == 0 {
		return errbuilder.NewErrBuilder().
			WithMsg("At least one subject is required").
			WithCode(errbuilder.CodeInvalidArgument)
	}

	// Create stream configuration
	config := StreamConfig{
		Subjects:         subjects,
		ProcessingEnabled: true,
		RetentionPolicy:   "limits",
		MaxAge:           86400, // 24 hours
		MaxBytes:         1073741824, // 1 GB
	}

	// Check if broker supports streams
	streamBroker, ok := s.broker.(brokers.StreamBroker)
	if !ok {
		return errbuilder.NewErrBuilder().
			WithMsg("Broker does not support streams").
			WithCode(errbuilder.CodeUnimplemented)
	}

	// Create stream
	ctx := context.Background()
	if err := streamBroker.CreateStream(ctx, name, config.Subjects); err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg("Failed to create stream").
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Store stream configuration
	s.streams[name] = config

	// If processing is enabled, subscribe to the stream
	if config.ProcessingEnabled {
		for _, subject := range config.Subjects {
			// Create a subscription for each subject
			_, err := s.broker.Subscribe(ctx, subject, func(ctx context.Context, msg *message.Message) error {
				// Process message through pipeline
				processedMsg, err := s.pipeline.Process(ctx, msg)
				if err != nil {
					s.logger.Error("Failed to process message", "error", err)
					return err
				}

				// Publish processed message to output subject
				outputSubject := subject + ".processed"
				if err := s.broker.Publish(ctx, outputSubject, processedMsg); err != nil {
					s.logger.Error("Failed to publish processed message", "error", err)
					return err
				}

				return nil
			})

			if err != nil {
				s.logger.Error("Failed to subscribe to subject", "subject", subject, "error", err)
				// Continue with other subjects even if one fails
			}
		}
	}

	s.logger.Info("Created stream", "name", name, "subjects", config.Subjects)
	return nil
}

// DeleteStream deletes a stream
func (s *Service) DeleteStream(name string) error {
	s.logger.Info("Deleting stream", "name", name)

	// Validate inputs
	if name == "" {
		return errbuilder.NewErrBuilder().
			WithMsg("Stream name is required").
			WithCode(errbuilder.CodeInvalidArgument)
	}

	// Check if stream exists
	s.mu.RLock()
	_, exists := s.streams[name]
	s.mu.RUnlock()

	if !exists {
		return errbuilder.NewErrBuilder().
			WithMsg("Stream not found").
			WithCode(errbuilder.CodeNotFound)
	}

	// Check if broker supports streams
	streamBroker, ok := s.broker.(brokers.StreamBroker)
	if !ok {
		return errbuilder.NewErrBuilder().
			WithMsg("Broker does not support streams").
			WithCode(errbuilder.CodeUnimplemented)
	}

	// Delete stream
	ctx := context.Background()
	if err := streamBroker.DeleteStream(ctx, name); err != nil {
		return errbuilder.NewErrBuilder().
			WithMsg("Failed to delete stream").
			WithCode(errbuilder.CodeInternal).
			WithCause(err)
	}

	// Remove stream configuration
	delete(s.streams, name)

	s.logger.Info("Deleted stream", "name", name)
	return nil
}

// GetStreamConfig retrieves the configuration for a stream
func (s *Service) GetStreamConfig(name string) (StreamConfig, bool) {
	config, exists := s.streams[name]
	return config, exists
}

// ListStreams lists all streams
func (s *Service) ListStreams() map[string]StreamConfig {
	return s.streams
}

// PublishToStream publishes a message to a stream
func (s *Service) PublishToStream(ctx context.Context, streamName string, subject string, msg *message.Message) error {
	// Check if stream exists
	if _, exists := s.streams[streamName]; !exists {
		return errbuilder.GenericErr("Stream does not exist", nil)
	}

	// Publish message to subject
	if err := s.broker.Publish(ctx, subject, msg); err != nil {
		return errbuilder.GenericErr("Failed to publish message to stream", err)
	}

	return nil
}

// SubscribeToStream subscribes to a stream
func (s *Service) SubscribeToStream(
	ctx context.Context,
	streamName string,
	subject string,
	handler brokers.MessageHandler,
) (brokers.Subscription, error) {
	// Check if stream exists
	if _, exists := s.streams[streamName]; !exists {
		return nil, errbuilder.GenericErr("Stream does not exist", nil)
	}

	// Subscribe to subject
	subscription, err := s.broker.Subscribe(ctx, subject, handler)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to subscribe to stream", err)
	}

	return subscription, nil
}
