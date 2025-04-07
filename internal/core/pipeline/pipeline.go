package pipeline

import (
	"context"
	"sync"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// ProcessingStage represents a stage in the processing pipeline
type ProcessingStage interface {
	// Process processes a message and returns the processed message
	Process(ctx context.Context, msg *message.Message) (*message.Message, error)

	// GetID returns the ID of the stage
	GetID() string

	// GetName returns the name of the stage
	GetName() string

	// GetType returns the type of the stage
	GetType() string

	// GetConfig returns the configuration of the stage
	GetConfig() map[string]string
}

// Pipeline represents a processing pipeline
type Pipeline struct {
	ID          string
	Name        string
	Stages      []ProcessingStage
	InputTopic  string
	OutputTopic string
	logger      logger.Logger
	mu          sync.RWMutex
}

// NewPipeline creates a new pipeline
func NewPipeline(id, name, inputTopic, outputTopic string, logger logger.Logger) *Pipeline {
	return &Pipeline{
		ID:          id,
		Name:        name,
		Stages:      make([]ProcessingStage, 0),
		InputTopic:  inputTopic,
		OutputTopic: outputTopic,
		logger:      logger.With("component", "pipeline", "pipeline_id", id),
	}
}

// AddStage adds a stage to the pipeline
func (p *Pipeline) AddStage(stage ProcessingStage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Stages = append(p.Stages, stage)
}

// RemoveStage removes a stage from the pipeline
func (p *Pipeline) RemoveStage(stageID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, stage := range p.Stages {
		if stage.GetID() == stageID {
			// Remove stage
			p.Stages = append(p.Stages[:i], p.Stages[i+1:]...)
			return nil
		}
	}

	return errbuilder.GenericErr("Stage not found", nil)
}

// Process processes a message through the pipeline
func (p *Pipeline) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	p.mu.RLock()
	stages := p.Stages
	p.mu.RUnlock()

	// Process message through each stage
	var err error
	currentMsg := msg
	for _, stage := range stages {
		p.logger.Debug("Processing message through stage", "stage", stage.GetName())
		currentMsg, err = stage.Process(ctx, currentMsg)
		if err != nil {
			return nil, errbuilder.GenericErr("Failed to process message", err)
		}
		if currentMsg == nil {
			// Stage filtered out the message
			p.logger.Debug("Message filtered out by stage", "stage", stage.GetName())
			return nil, nil
		}
	}

	return currentMsg, nil
}

// GetStages returns the stages in the pipeline
func (p *Pipeline) GetStages() []ProcessingStage {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Stages
}

// BaseStage provides common functionality for processing stages
type BaseStage struct {
	ID     string
	Name   string
	Type   string
	Config map[string]string
}

// GetID returns the ID of the stage
func (s *BaseStage) GetID() string {
	return s.ID
}

// GetName returns the name of the stage
func (s *BaseStage) GetName() string {
	return s.Name
}

// GetType returns the type of the stage
func (s *BaseStage) GetType() string {
	return s.Type
}

// GetConfig returns the configuration of the stage
func (s *BaseStage) GetConfig() map[string]string {
	return s.Config
}

// ValidationStage validates messages
type ValidationStage struct {
	BaseStage
	Validator func(ctx context.Context, msg *message.Message) error
}

// Process validates a message
func (s *ValidationStage) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	if s.Validator != nil {
		if err := s.Validator(ctx, msg); err != nil {
			return nil, errbuilder.GenericErr("Validation failed", err)
		}
	}
	return msg, nil
}

// TransformationStage transforms messages
type TransformationStage struct {
	BaseStage
	Transformer func(ctx context.Context, msg *message.Message) (*message.Message, error)
}

// Process transforms a message
func (s *TransformationStage) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	if s.Transformer != nil {
		return s.Transformer(ctx, msg)
	}
	return msg, nil
}

// EnrichmentStage enriches messages with additional data
type EnrichmentStage struct {
	BaseStage
	Enricher func(ctx context.Context, msg *message.Message) (*message.Message, error)
}

// Process enriches a message
func (s *EnrichmentStage) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	if s.Enricher != nil {
		return s.Enricher(ctx, msg)
	}
	return msg, nil
}

// FilterStage filters messages
type FilterStage struct {
	BaseStage
	Filter func(ctx context.Context, msg *message.Message) (bool, error)
}

// Process filters a message
func (s *FilterStage) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	if s.Filter != nil {
		pass, err := s.Filter(ctx, msg)
		if err != nil {
			return nil, errbuilder.GenericErr("Filter error", err)
		}
		if !pass {
			return nil, nil // Filter out the message
		}
	}
	return msg, nil
}

// RoutingStage routes messages to different topics
type RoutingStage struct {
	BaseStage
	Router func(ctx context.Context, msg *message.Message) (string, error)
}

// Process routes a message
func (s *RoutingStage) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	if s.Router != nil {
		topic, err := s.Router(ctx, msg)
		if err != nil {
			return nil, errbuilder.GenericErr("Routing error", err)
		}
		// Set the target topic in the message metadata
		msg.SetMetadata("target_topic", topic)
	}
	return msg, nil
}
