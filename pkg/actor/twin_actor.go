package actor

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// TwinActor represents a digital twin actor
type TwinActor struct {
	twinID      string
	devicePID   *actor.PID
	state       map[string]any
	desired     map[string]any
	lastUpdated time.Time
	logger      logger.Logger
	mu          sync.RWMutex
}

// NewTwinActor creates a new twin actor
func NewTwinActor(twinID string, devicePID *actor.PID, logger logger.Logger) *TwinActor {
	return &TwinActor{
		twinID:      twinID,
		devicePID:   devicePID,
		state:       make(map[string]any),
		desired:     make(map[string]any),
		lastUpdated: time.Now(),
		logger:      logger.With("actor", "twin", "twin_id", twinID),
	}
}

// Receive handles incoming messages
func (a *TwinActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *message.Message:
		a.handleMessage(ctx, msg)
	default:
		a.logger.Debug("Unknown message type", "type", ctx.Message())
	}
}

// handleMessage handles incoming messages
func (a *TwinActor) handleMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling message", "message_id", msg.ID, "type", msg.Type)

	switch msg.Type {
	case message.TypeTelemetry:
		// Handle telemetry message
		a.handleTelemetryMessage(ctx, msg)
	case message.TypeEvent:
		// Handle event message
		a.handleEventMessage(ctx, msg)
	case message.TypeCommand:
		// Handle command message
		a.handleCommandMessage(ctx, msg)
	default:
		a.logger.Warn("Unknown message type", "type", msg.Type)
	}
}

// handleTelemetryMessage handles telemetry messages
func (a *TwinActor) handleTelemetryMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling telemetry message", "message_id", msg.ID)

	// Parse telemetry data
	var telemetry map[string]any
	if err := json.Unmarshal(msg.Payload, &telemetry); err != nil {
		a.logger.Error("Failed to parse telemetry payload", "error", err)
		return
	}

	// Update state
	a.mu.Lock()
	for k, v := range telemetry {
		a.state[k] = v
	}
	a.lastUpdated = time.Now()
	a.mu.Unlock()

	// Check for delta between desired and current state
	a.checkDesiredStateDelta(ctx)
}

// handleEventMessage handles event messages
func (a *TwinActor) handleEventMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling event message", "message_id", msg.ID)

	// Parse event payload
	var event struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		a.logger.Error("Failed to parse event payload", "error", err)
		return
	}

	// Process event based on type
	switch event.Type {
	case "state_changed":
		// Update state
		a.mu.Lock()
		for k, v := range event.Payload {
			a.state[k] = v
		}
		a.lastUpdated = time.Now()
		a.mu.Unlock()

		// Check for delta between desired and current state
		a.checkDesiredStateDelta(ctx)

	default:
		a.logger.Warn("Unknown event type", "type", event.Type)
	}
}

// handleCommandMessage handles command messages
func (a *TwinActor) handleCommandMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling command message", "message_id", msg.ID)

	// Parse command payload
	var command struct {
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}

	if err := json.Unmarshal(msg.Payload, &command); err != nil {
		a.logger.Error("Failed to parse command payload", "error", err)
		a.sendErrorResponse(ctx, msg, "invalid_command", "Failed to parse command payload")
		return
	}

	// Process command based on action
	var responsePayload []byte
	var err error

	switch command.Action {
	case "update_desired":
		// Update desired state
		a.mu.Lock()
		for k, v := range command.Params {
			a.desired[k] = v
		}
		a.mu.Unlock()

		// Create response payload
		responsePayload, err = json.Marshal(map[string]any{
			"status":  "success",
			"desired": a.desired,
		})

		// Check for delta between desired and current state
		a.checkDesiredStateDelta(ctx)

	case "get_state":
		// Read state
		a.mu.RLock()
		state := make(map[string]any, len(a.state))
		for k, v := range a.state {
			state[k] = v
		}
		a.mu.RUnlock()

		// Create response payload
		responsePayload, err = json.Marshal(map[string]any{
			"status": "success",
			"state":  state,
		})

	case "get_desired":
		// Read desired state
		a.mu.RLock()
		desired := make(map[string]any, len(a.desired))
		for k, v := range a.desired {
			desired[k] = v
		}
		a.mu.RUnlock()

		// Create response payload
		responsePayload, err = json.Marshal(map[string]any{
			"status":  "success",
			"desired": desired,
		})

	default:
		a.logger.Warn("Unknown command action", "action", command.Action)
		a.sendErrorResponse(ctx, msg, "unknown_action", "Unknown command action: "+command.Action)
		return
	}

	if err != nil {
		a.logger.Error("Failed to create response payload", "error", err)
		a.sendErrorResponse(ctx, msg, "internal_error", "Failed to process command")
		return
	}

	// Send response
	response := message.NewMessage(
		uuid.New().String(),
		a.twinID,
		msg.Source,
		message.TypeResponse,
		responsePayload,
	)
	response.SetContentType("application/json")

	// Send response to the original sender
	if msg.Source != "" {
		ctx.Send(actor.NewPID("local", msg.Source), response)
	}
}

// sendErrorResponse sends an error response to the original sender
func (a *TwinActor) sendErrorResponse(ctx actor.Context, originalMsg *message.Message, code string, message string) {
	payload, _ := json.Marshal(map[string]any{
		"status":  "error",
		"code":    code,
		"message": message,
	})

	response := message.NewMessage(
		uuid.New().String(),
		a.twinID,
		originalMsg.Source,
		message.TypeResponse,
		payload,
	)
	response.SetContentType("application/json")

	// Send response to the original sender
	if originalMsg.Source != "" {
		ctx.Send(actor.NewPID("local", originalMsg.Source), response)
	}
}

// checkDesiredStateDelta checks for differences between desired and current state
func (a *TwinActor) checkDesiredStateDelta(ctx actor.Context) {
	// Skip if no device PID
	if a.devicePID == nil {
		return
	}

	// Get current and desired state
	a.mu.RLock()
	state := make(map[string]any, len(a.state))
	for k, v := range a.state {
		state[k] = v
	}
	desired := make(map[string]any, len(a.desired))
	for k, v := range a.desired {
		desired[k] = v
	}
	a.mu.RUnlock()

	// Check for differences
	delta := make(map[string]any)
	for k, v := range desired {
		if currentVal, ok := state[k]; !ok || !equals(currentVal, v) {
			delta[k] = v
		}
	}

	// If there are differences, send a command to the device
	if len(delta) > 0 {
		a.logger.Info("Detected state delta", "delta", delta)

		// Create command payload
		commandPayload, _ := json.Marshal(map[string]any{
			"action": "set_state",
			"params": delta,
		})

		// Create command message
		command := message.NewMessage(
			uuid.New().String(),
			a.twinID,
			"device:"+a.twinID,
			message.TypeCommand,
			commandPayload,
		)
		command.SetContentType("application/json")

		// Send command to device
		ctx.Send(a.devicePID, command)
	}
}

// equals compares two values for equality
func equals(a, b any) bool {
	// For simplicity, we'll just compare the JSON representation
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
