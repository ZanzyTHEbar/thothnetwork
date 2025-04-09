package actor

import (
	"fmt"
	"slices"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/anthdm/hollywood/actor"
)

// PipelineActor represents a processing pipeline in the system
type PipelineActor struct {
	pipelineID string
	stages     []*actor.PID
	logger     logger.Logger
}

// NewPipelineActor creates a new pipeline actor
func NewPipelineActor(pipelineID string, log logger.Logger) *PipelineActor {
	return &PipelineActor{
		pipelineID: pipelineID,
		stages:     make([]*actor.PID, 0),
		logger:     log.With("actor", "pipeline", "pipeline_id", pipelineID),
	}
}

// Receive handles messages sent to the pipeline actor
func (a *PipelineActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.handleStarted(ctx)
	case *actor.Stopped:
		a.handleStopped(ctx)
	case *message.Message:
		a.handleMessage(ctx, msg)
	case *AddStageCommand:
		a.handleAddStage(ctx, msg)
	case *RemoveStageCommand:
		a.handleRemoveStage(ctx, msg)
	case *GetStagesQuery:
		a.handleGetStages(ctx, msg)
	default:
		a.logger.Warn("Received unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

// handleStarted handles the actor started event
func (a *PipelineActor) handleStarted(_ actor.Context) {
	a.logger.Info("Pipeline actor started")
}

// handleStopped handles the actor stopped event
func (a *PipelineActor) handleStopped(_ actor.Context) {
	a.logger.Info("Pipeline actor stopped")
}

// handleMessage handles messages sent to the pipeline
func (a *PipelineActor) handleMessage(_ actor.Context, msg *message.Message) {
	a.logger.Info("Received message", "message_id", msg.ID, "message_type", msg.Type)

	// Process message through pipeline stages
	if len(a.stages) > 0 {
		// In the current version, we would need to handle pipeline stages differently
		// For now, we'll just log the pipeline processing
		a.logger.Info("Processing message through pipeline", "stages", len(a.stages))
	}
}

// handleAddStage handles add stage commands
func (a *PipelineActor) handleAddStage(ctx actor.Context, cmd *AddStageCommand) {
	a.logger.Debug("Adding stage to pipeline", "stage_id", cmd.StageID)

	// Add stage to pipeline
	a.stages = append(a.stages, cmd.StagePID)

	// Send response
	ctx.Respond(&AddStageResponse{Success: true})
}

// handleRemoveStage handles remove stage commands
func (a *PipelineActor) handleRemoveStage(ctx actor.Context, cmd *RemoveStageCommand) {
	a.logger.Debug("Removing stage from pipeline", "stage_id", cmd.StageID)

	// Find stage
	for i, stagePID := range a.stages {
		if stagePID.ID == cmd.StageID {
			// Remove stage
			a.stages = slices.Delete(a.stages, i, i+1)
			break
		}
	}

	// Send response
	ctx.Respond(&RemoveStageResponse{Success: true})
}

// handleGetStages handles get stages queries
func (a *PipelineActor) handleGetStages(ctx actor.Context, _ *GetStagesQuery) {
	a.logger.Debug("Getting stages in pipeline")

	// Get stages
	stageIDs := make([]string, 0, len(a.stages))
	for _, stagePID := range a.stages {
		stageIDs = append(stageIDs, stagePID.ID)
	}

	// Send response
	ctx.Respond(&GetStagesResponse{StageIDs: stageIDs})
}
