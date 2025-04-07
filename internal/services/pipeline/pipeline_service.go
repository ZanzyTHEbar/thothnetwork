package pipeline

import (
	"context"
	"sync"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/pipeline"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/brokers"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Service manages pipelines
type Service struct {
	pipelines     map[string]*pipeline.Pipeline
	broker        brokers.MessageBroker
	logger        logger.Logger
	subscriptions map[string]brokers.Subscription
	mu            sync.RWMutex
}

// NewService creates a new pipeline service
func NewService(broker brokers.MessageBroker, logger logger.Logger) *Service {
	return &Service{
		pipelines:     make(map[string]*pipeline.Pipeline),
		broker:        broker,
		logger:        logger.With("component", "pipeline_service"),
		subscriptions: make(map[string]brokers.Subscription),
	}
}

// CreatePipeline creates a new pipeline
func (s *Service) CreatePipeline(ctx context.Context, name, inputTopic, outputTopic string, stages []pipeline.ProcessingStage) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate pipeline ID
	pipelineID := uuid.New().String()

	// Create pipeline
	p := pipeline.NewPipeline(pipelineID, name, inputTopic, outputTopic, s.logger)

	// Add stages
	for _, stage := range stages {
		p.AddStage(stage)
	}

	// Store pipeline
	s.pipelines[pipelineID] = p

	s.logger.Info("Pipeline created", "id", pipelineID, "name", name)
	return pipelineID, nil
}

// GetPipeline retrieves a pipeline by ID
func (s *Service) GetPipeline(ctx context.Context, pipelineID string) (*pipeline.Pipeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, exists := s.pipelines[pipelineID]
	if !exists {
		return nil, errbuilder.GenericErr("Pipeline not found", nil)
	}

	return p, nil
}

// UpdatePipeline updates an existing pipeline
func (s *Service) UpdatePipeline(ctx context.Context, pipelineID, name, inputTopic, outputTopic string, stages []pipeline.ProcessingStage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if pipeline exists
	p, exists := s.pipelines[pipelineID]
	if !exists {
		return errbuilder.GenericErr("Pipeline not found", nil)
	}

	// Stop pipeline if running
	if sub, running := s.subscriptions[pipelineID]; running {
		if err := sub.Unsubscribe(); err != nil {
			return errbuilder.GenericErr("Failed to stop pipeline", err)
		}
		delete(s.subscriptions, pipelineID)
	}

	// Update pipeline
	p.Name = name
	p.InputTopic = inputTopic
	p.OutputTopic = outputTopic

	// Replace stages
	p.Stages = make([]pipeline.ProcessingStage, 0)
	for _, stage := range stages {
		p.AddStage(stage)
	}

	s.logger.Info("Pipeline updated", "id", pipelineID, "name", name)
	return nil
}

// DeletePipeline deletes a pipeline
func (s *Service) DeletePipeline(ctx context.Context, pipelineID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if pipeline exists
	if _, exists := s.pipelines[pipelineID]; !exists {
		return errbuilder.GenericErr("Pipeline not found", nil)
	}

	// Stop pipeline if running
	if sub, running := s.subscriptions[pipelineID]; running {
		if err := sub.Unsubscribe(); err != nil {
			return errbuilder.GenericErr("Failed to stop pipeline", err)
		}
		delete(s.subscriptions, pipelineID)
	}

	// Delete pipeline
	delete(s.pipelines, pipelineID)

	s.logger.Info("Pipeline deleted", "id", pipelineID)
	return nil
}

// ListPipelines lists all pipelines
func (s *Service) ListPipelines(ctx context.Context) []*pipeline.Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pipelines := make([]*pipeline.Pipeline, 0, len(s.pipelines))
	for _, p := range s.pipelines {
		pipelines = append(pipelines, p)
	}

	return pipelines
}

// StartPipeline starts a pipeline
func (s *Service) StartPipeline(ctx context.Context, pipelineID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if pipeline exists
	p, exists := s.pipelines[pipelineID]
	if !exists {
		return errbuilder.GenericErr("Pipeline not found", nil)
	}

	// Check if pipeline is already running
	if _, running := s.subscriptions[pipelineID]; running {
		return errbuilder.GenericErr("Pipeline already running", nil)
	}

	// Create message handler
	handler := func(ctx context.Context, msg *message.Message) error {
		return s.processPipelineMessage(ctx, p, msg)
	}

	// Subscribe to input topic
	sub, err := s.broker.Subscribe(ctx, p.InputTopic, handler)
	if err != nil {
		return errbuilder.GenericErr("Failed to subscribe to input topic", err)
	}

	// Store subscription
	s.subscriptions[pipelineID] = sub

	s.logger.Info("Pipeline started", "id", pipelineID, "name", p.Name)
	return nil
}

// StopPipeline stops a pipeline
func (s *Service) StopPipeline(ctx context.Context, pipelineID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if pipeline exists
	if _, exists := s.pipelines[pipelineID]; !exists {
		return errbuilder.GenericErr("Pipeline not found", nil)
	}

	// Check if pipeline is running
	sub, running := s.subscriptions[pipelineID]
	if !running {
		return errbuilder.GenericErr("Pipeline not running", nil)
	}

	// Unsubscribe from input topic
	if err := sub.Unsubscribe(); err != nil {
		return errbuilder.GenericErr("Failed to unsubscribe from input topic", err)
	}

	// Remove subscription
	delete(s.subscriptions, pipelineID)

	s.logger.Info("Pipeline stopped", "id", pipelineID)
	return nil
}

// ProcessMessage processes a message through a pipeline
func (s *Service) ProcessMessage(ctx context.Context, pipelineID string, msg *message.Message) (*message.Message, error) {
	s.mu.RLock()
	p, exists := s.pipelines[pipelineID]
	s.mu.RUnlock()

	if !exists {
		return nil, errbuilder.GenericErr("Pipeline not found", nil)
	}

	// Process message through pipeline
	return p.Process(ctx, msg)
}

// processPipelineMessage processes a message through a pipeline and publishes the result
func (s *Service) processPipelineMessage(ctx context.Context, p *pipeline.Pipeline, msg *message.Message) error {
	// Process message through pipeline
	result, err := p.Process(ctx, msg)
	if err != nil {
		s.logger.Error("Failed to process message", "error", err)
		return err
	}

	// Check if message was filtered out
	if result == nil {
		return nil
	}

	// Check if message has a target topic in metadata
	targetTopic := p.OutputTopic
	if topic, exists := result.Metadata["target_topic"]; exists {
		targetTopic = topic
	}

	// Publish result to output topic
	if err := s.broker.Publish(ctx, targetTopic, result); err != nil {
		s.logger.Error("Failed to publish message", "error", err)
		return err
	}

	return nil
}

// AddStage adds a stage to a pipeline
func (s *Service) AddStage(ctx context.Context, pipelineID string, stage pipeline.ProcessingStage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if pipeline exists
	p, exists := s.pipelines[pipelineID]
	if !exists {
		return errbuilder.GenericErr("Pipeline not found", nil)
	}

	// Add stage
	p.AddStage(stage)

	s.logger.Info("Stage added to pipeline", "pipeline_id", pipelineID, "stage_id", stage.GetID())
	return nil
}

// RemoveStage removes a stage from a pipeline
func (s *Service) RemoveStage(ctx context.Context, pipelineID, stageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if pipeline exists
	p, exists := s.pipelines[pipelineID]
	if !exists {
		return errbuilder.GenericErr("Pipeline not found", nil)
	}

	// Remove stage
	if err := p.RemoveStage(stageID); err != nil {
		return err
	}

	s.logger.Info("Stage removed from pipeline", "pipeline_id", pipelineID, "stage_id", stageID)
	return nil
}

// CreateValidationStage creates a validation stage
func (s *Service) CreateValidationStage(name string, validator func(ctx context.Context, msg *message.Message) error) pipeline.ProcessingStage {
	return &pipeline.ValidationStage{
		BaseStage: pipeline.BaseStage{
			ID:     uuid.New().String(),
			Name:   name,
			Type:   "validation",
			Config: make(map[string]string),
		},
		Validator: validator,
	}
}

// CreateTransformationStage creates a transformation stage
func (s *Service) CreateTransformationStage(name string, transformer func(ctx context.Context, msg *message.Message) (*message.Message, error)) pipeline.ProcessingStage {
	return &pipeline.TransformationStage{
		BaseStage: pipeline.BaseStage{
			ID:     uuid.New().String(),
			Name:   name,
			Type:   "transformation",
			Config: make(map[string]string),
		},
		Transformer: transformer,
	}
}

// CreateEnrichmentStage creates an enrichment stage
func (s *Service) CreateEnrichmentStage(name string, enricher func(ctx context.Context, msg *message.Message) (*message.Message, error)) pipeline.ProcessingStage {
	return &pipeline.EnrichmentStage{
		BaseStage: pipeline.BaseStage{
			ID:     uuid.New().String(),
			Name:   name,
			Type:   "enrichment",
			Config: make(map[string]string),
		},
		Enricher: enricher,
	}
}

// CreateFilterStage creates a filter stage
func (s *Service) CreateFilterStage(name string, filter func(ctx context.Context, msg *message.Message) (bool, error)) pipeline.ProcessingStage {
	return &pipeline.FilterStage{
		BaseStage: pipeline.BaseStage{
			ID:     uuid.New().String(),
			Name:   name,
			Type:   "filter",
			Config: make(map[string]string),
		},
		Filter: filter,
	}
}

// CreateRoutingStage creates a routing stage
func (s *Service) CreateRoutingStage(name string, router func(ctx context.Context, msg *message.Message) (string, error)) pipeline.ProcessingStage {
	return &pipeline.RoutingStage{
		BaseStage: pipeline.BaseStage{
			ID:     uuid.New().String(),
			Name:   name,
			Type:   "routing",
			Config: make(map[string]string),
		},
		Router: router,
	}
}
