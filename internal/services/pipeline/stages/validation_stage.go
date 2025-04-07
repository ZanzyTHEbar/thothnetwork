package stages

import (
	"context"
	"slices"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Validator defines a function that validates a message
type Validator func(ctx context.Context, msg *message.Message) error

// ValidationStage is a pipeline stage that validates messages
type ValidationStage struct {
	id         string
	name       string
	validators []Validator
	logger     logger.Logger
}

// NewValidationStage creates a new validation stage
func NewValidationStage(name string, logger logger.Logger) *ValidationStage {
	return &ValidationStage{
		id:         uuid.New().String(),
		name:       name,
		validators: make([]Validator, 0),
		logger:     logger.With("stage", "validation", "name", name),
	}
}

// ID returns the unique identifier for the stage
func (s *ValidationStage) ID() string {
	return s.id
}

// Name returns the name of the stage
func (s *ValidationStage) Name() string {
	return s.name
}

// AddValidator adds a validator to the stage
func (s *ValidationStage) AddValidator(validator Validator) {
	s.validators = append(s.validators, validator)
}

// Process processes a message through all validators
func (s *ValidationStage) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	// Run all validators
	for _, validator := range s.validators {
		if err := validator(ctx, msg); err != nil {
			s.logger.Warn("Message validation failed", "error", err)
			return nil, errbuilder.GenericErr("Message validation failed", err)
		}
	}

	return msg, nil
}

// Common validators

// RequiredFieldsValidator validates that required fields are present
func RequiredFieldsValidator() Validator {
	return func(ctx context.Context, msg *message.Message) error {
		if msg.ID == "" {
			return errbuilder.GenericErr("Message ID is required", nil)
		}

		if msg.Source == "" {
			return errbuilder.GenericErr("Message source is required", nil)
		}

		if msg.Type == "" {
			return errbuilder.GenericErr("Message type is required", nil)
		}

		return nil
	}
}

// PayloadSizeValidator validates that the payload size is within limits
func PayloadSizeValidator(maxSize int) Validator {
	return func(ctx context.Context, msg *message.Message) error {
		if len(msg.Payload) > maxSize {
			return errbuilder.GenericErr("Payload size exceeds maximum", nil)
		}

		return nil
	}
}

// ContentTypeValidator validates that the content type is supported
func ContentTypeValidator(supportedTypes ...string) Validator {
	return func(ctx context.Context, msg *message.Message) error {
		if msg.ContentType == "" {
			return nil // No content type specified, skip validation
		}

		if slices.Contains(supportedTypes, msg.ContentType) {
			return nil
		}

		return errbuilder.GenericErr("Unsupported content type", nil)
	}
}
