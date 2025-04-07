package stages

import (
	"context"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Transformer defines a function that transforms a message
type Transformer func(ctx context.Context, msg *message.Message) (*message.Message, error)

// TransformationStage is a pipeline stage that transforms messages
type TransformationStage struct {
	id           string
	name         string
	transformers []Transformer
	logger       logger.Logger
}

// NewTransformationStage creates a new transformation stage
func NewTransformationStage(name string, logger logger.Logger) *TransformationStage {
	return &TransformationStage{
		id:           uuid.New().String(),
		name:         name,
		transformers: make([]Transformer, 0),
		logger:       logger.With("stage", "transformation", "name", name),
	}
}

// ID returns the unique identifier for the stage
func (s *TransformationStage) ID() string {
	return s.id
}

// Name returns the name of the stage
func (s *TransformationStage) Name() string {
	return s.name
}

// AddTransformer adds a transformer to the stage
func (s *TransformationStage) AddTransformer(transformer Transformer) {
	s.transformers = append(s.transformers, transformer)
}

// Process processes a message through all transformers
func (s *TransformationStage) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	currentMsg := msg

	// Run all transformers
	for _, transformer := range s.transformers {
		var err error
		currentMsg, err = transformer(ctx, currentMsg)
		if err != nil {
			s.logger.Warn("Message transformation failed", "error", err)
			return nil, errbuilder.GenericErr("Message transformation failed", err)
		}

		// If the transformer returns nil, stop processing
		if currentMsg == nil {
			return nil, nil
		}
	}

	return currentMsg, nil
}

// Common transformers

// ContentTypeTransformer transforms the content type of a message
func ContentTypeTransformer(sourceType, targetType string, transformFn func([]byte) ([]byte, error)) Transformer {
	return func(ctx context.Context, msg *message.Message) (*message.Message, error) {
		// Only transform messages with the source content type
		if msg.ContentType != sourceType {
			return msg, nil
		}

		// Transform payload
		newPayload, err := transformFn(msg.Payload)
		if err != nil {
			return nil, errbuilder.GenericErr("Failed to transform payload", err)
		}

		// Create a new message with the transformed payload
		newMsg := *msg
		newMsg.Payload = newPayload
		newMsg.ContentType = targetType

		return &newMsg, nil
	}
}

// MetadataTransformer adds or modifies metadata in a message
func MetadataTransformer(metadataFn func(map[string]string) map[string]string) Transformer {
	return func(ctx context.Context, msg *message.Message) (*message.Message, error) {
		// Create a new message with the transformed metadata
		newMsg := *msg

		// Initialize metadata if nil
		if newMsg.Metadata == nil {
			newMsg.Metadata = make(map[string]string)
		}

		// Transform metadata
		newMsg.Metadata = metadataFn(newMsg.Metadata)

		return &newMsg, nil
	}
}

// FilterTransformer filters messages based on a predicate
func FilterTransformer(predicate func(*message.Message) bool) Transformer {
	return func(ctx context.Context, msg *message.Message) (*message.Message, error) {
		if predicate(msg) {
			return msg, nil
		}

		// Return nil to drop the message
		return nil, nil
	}
}
