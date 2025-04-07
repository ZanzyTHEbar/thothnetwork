package pipeline

import (
	"context"
	"sync"

	"github.com/ZanzyTHEbar/errbuilder-go"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/worker"
)

// ProcessingStage represents a stage in the processing pipeline
type ProcessingStage interface {
	// Process processes a message and returns the processed message
	Process(ctx context.Context, msg *message.Message) (*message.Message, error)

	// ID returns the unique identifier for the stage
	ID() string

	// Name returns the name of the stage
	Name() string
}

// Pipeline represents a processing pipeline for messages
type Pipeline struct {
	stages     []ProcessingStage
	workerPool *worker.Pool
	logger     logger.Logger
	mu         sync.RWMutex
}

// NewPipeline creates a new processing pipeline
func NewPipeline(workerPool *worker.Pool, logger logger.Logger) *Pipeline {
	return &Pipeline{
		stages:     make([]ProcessingStage, 0),
		workerPool: workerPool,
		logger:     logger.With("component", "pipeline"),
		mu:         sync.RWMutex{},
	}
}

// AddStage adds a processing stage to the pipeline
func (p *Pipeline) AddStage(stage ProcessingStage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if stage with same ID already exists
	for _, s := range p.stages {
		if s.ID() == stage.ID() {
			return errbuilder.New().
				WithMessage("Stage with same ID already exists").
				WithField("stage_id", stage.ID()).
				Build()
		}
	}

	p.stages = append(p.stages, stage)
	p.logger.Info("Added processing stage", "stage_id", stage.ID(), "stage_name", stage.Name())
	return nil
}

// RemoveStage removes a processing stage from the pipeline
func (p *Pipeline) RemoveStage(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, stage := range p.stages {
		if stage.ID() == id {
			p.stages = append(p.stages[:i], p.stages[i+1:]...)
			p.logger.Info("Removed processing stage", "stage_id", id)
			return nil
		}
	}

	return errbuilder.New().
		WithMessage("Stage not found").
		WithField("stage_id", id).
		Build()
}

// GetStages returns all processing stages in the pipeline
func (p *Pipeline) GetStages() []ProcessingStage {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy of the stages slice
	stages := make([]ProcessingStage, len(p.stages))
	copy(stages, p.stages)
	return stages
}

// Process processes a message through all stages in the pipeline
func (p *Pipeline) Process(ctx context.Context, msg *message.Message) (*message.Message, error) {
	p.mu.RLock()
	stages := make([]ProcessingStage, len(p.stages))
	copy(stages, p.stages)
	p.mu.RUnlock()

	// If there are no stages, return the original message
	if len(stages) == 0 {
		return msg, nil
	}

	// Process message through each stage
	currentMsg := msg
	for _, stage := range stages {
		var err error

		// Process the message
		currentMsg, err = stage.Process(ctx, currentMsg)
		if err != nil {
			return nil, errbuilder.New().
				WithMessage("Failed to process message").
				WithField("stage_id", stage.ID()).
				WithField("stage_name", stage.Name()).
				WithError(err).
				Build()
		}

		// If the message is nil, stop processing
		if currentMsg == nil {
			p.logger.Debug("Message dropped by stage", "stage_id", stage.ID(), "stage_name", stage.Name())
			return nil, nil
		}
	}

	return currentMsg, nil
}

// ProcessAsync processes a message asynchronously through all stages in the pipeline
func (p *Pipeline) ProcessAsync(ctx context.Context, msg *message.Message, callback func(*message.Message, error)) {
	p.workerPool.Submit(func() {
		result, err := p.Process(ctx, msg)
		callback(result, err)
	})
}
