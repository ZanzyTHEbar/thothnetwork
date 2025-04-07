package stream

import (
	"context"

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
	messageBroker brokers.MessageBroker
	pipeline      *pipeline.Pipeline
	logger        logger.Logger
	streams       map[string]StreamConfig
}

// NewService creates a new stream service
func NewService(
	messageBroker brokers.MessageBroker,
	pipeline *pipeline.Pipeline,
	logger logger.Logger,
) *Service {
	return &Service{
		messageBroker: messageBroker,
		pipeline:      pipeline,
		logger:        logger.With("service", "stream"),
		streams:       make(map[string]StreamConfig),
	}
}

// CreateStream creates a new stream
func (s *Service) CreateStream(ctx context.Context, name string, config StreamConfig) error {
	// Create stream in message broker
	if err := s.messageBroker.CreateStream(ctx, name, config.Subjects); err != nil {
		return errbuilder.GenericErr("Failed to create stream", err)
	}

	// Store stream configuration
	s.streams[name] = config

	// If processing is enabled, subscribe to the stream
	if config.ProcessingEnabled {
		for _, subject := range config.Subjects {
			// Create a subscription for each subject
			_, err := s.messageBroker.Subscribe(ctx, subject, func(ctx context.Context, msg *message.Message) error {
				// Process message through pipeline
				processedMsg, err := s.pipeline.Process(ctx, msg)
				if err != nil {
					s.logger.Error("Failed to process message", "error", err)
					return err
				}

				// Publish processed message to output subject
				outputSubject := subject + ".processed"
				if err := s.messageBroker.Publish(ctx, outputSubject, processedMsg); err != nil {
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
func (s *Service) DeleteStream(ctx context.Context, name string) error {
	// Delete stream from message broker
	if err := s.messageBroker.DeleteStream(ctx, name); err != nil {
		return errbuilder.GenericErr("Failed to delete stream", err)
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
	if err := s.messageBroker.Publish(ctx, subject, msg); err != nil {
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
	subscription, err := s.messageBroker.Subscribe(ctx, subject, handler)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to subscribe to stream", err)
	}

	return subscription, nil
}
