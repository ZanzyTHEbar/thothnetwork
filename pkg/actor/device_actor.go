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

// DeviceActor represents a device actor
type DeviceActor struct {
	deviceID    string
	twinPID     *actor.PID
	state       map[string]any
	lastUpdated time.Time
	logger      logger.Logger
	mu          sync.RWMutex
	thresholds  map[string]Threshold
}

// Threshold defines a threshold for a telemetry value
type Threshold struct {
	Min              float64
	Max              float64
	AlertOnViolation bool
}

// NewDeviceActor creates a new device actor
func NewDeviceActor(deviceID string, logger logger.Logger) *DeviceActor {
	return &DeviceActor{
		deviceID:    deviceID,
		state:       make(map[string]any),
		lastUpdated: time.Now(),
		logger:      logger.With("actor", "device", "device_id", deviceID),
		thresholds:  make(map[string]Threshold),
	}
}

// SetTwinPID sets the PID of the associated digital twin
func (a *DeviceActor) SetTwinPID(pid *actor.PID) {
	a.twinPID = pid
}

// Receive handles incoming messages
func (a *DeviceActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *message.Message:
		a.handleMessage(ctx, msg)
	default:
		a.logger.Debug("Unknown message type", "type", ctx.Message())
	}
}

// handleMessage handles incoming messages
func (a *DeviceActor) handleMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling message", "message_id", msg.ID, "type", msg.Type)

	switch msg.Type {
	case message.TypeCommand:
		// Handle command message
		a.handleCommandMessage(ctx, msg)
	case message.TypeEvent:
		// Handle event message
		a.handleEventMessage(ctx, msg)
	case message.TypeTelemetry:
		// Handle telemetry message
		a.handleTelemetryMessage(ctx, msg)
	default:
		a.logger.Warn("Unknown message type", "type", msg.Type)
	}
}

// handleCommandMessage handles command messages
func (a *DeviceActor) handleCommandMessage(ctx actor.Context, msg *message.Message) {
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
	case "set_state":
		// Update device state
		a.mu.Lock()
		for k, v := range command.Params {
			a.state[k] = v
		}
		a.lastUpdated = time.Now()
		a.mu.Unlock()

		// Create response payload
		responsePayload, err = json.Marshal(map[string]any{
			"status": "success",
			"state":  a.state,
		})

	case "get_state":
		// Read device state
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

	case "reboot":
		// Simulate device reboot
		a.mu.Lock()
		a.state["status"] = "rebooting"
		a.lastUpdated = time.Now()
		a.mu.Unlock()

		// Create response payload
		responsePayload, err = json.Marshal(map[string]any{
			"status":  "success",
			"message": "Device is rebooting",
		})

		// Schedule a status update after "reboot"
		go func() {
			time.Sleep(5 * time.Second)
			a.mu.Lock()
			a.state["status"] = "online"
			a.lastUpdated = time.Now()
			a.mu.Unlock()
		}()

	case "set_threshold":
		// Set threshold for a telemetry value
		if len(command.Params) < 3 {
			a.sendErrorResponse(ctx, msg, "invalid_params", "Missing required parameters: key, min, max")
			return
		}

		key, ok := command.Params["key"].(string)
		if !ok {
			a.sendErrorResponse(ctx, msg, "invalid_params", "Key must be a string")
			return
		}

		min, ok := command.Params["min"].(float64)
		if !ok {
			a.sendErrorResponse(ctx, msg, "invalid_params", "Min must be a number")
			return
		}

		max, ok := command.Params["max"].(float64)
		if !ok {
			a.sendErrorResponse(ctx, msg, "invalid_params", "Max must be a number")
			return
		}

		alertOnViolation := true
		if alert, ok := command.Params["alert_on_violation"].(bool); ok {
			alertOnViolation = alert
		}

		// Set threshold
		a.mu.Lock()
		a.thresholds[key] = Threshold{
			Min:              min,
			Max:              max,
			AlertOnViolation: alertOnViolation,
		}
		a.mu.Unlock()

		// Create response payload
		responsePayload, err = json.Marshal(map[string]any{
			"status":  "success",
			"message": "Threshold set",
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
	response := message.New(
		uuid.New().String(),
		a.deviceID,
		msg.Source,
		message.TypeResponse,
		responsePayload,
	)
	response.SetContentType("application/json")

	// Send response to the original sender
	if msg.Source != "" {
		ctx.Send(actor.NewPID("local", msg.Source), response)
	}

	// If we have a twin, update it with the new state
	if a.twinPID != nil {
		a.mu.RLock()
		state := make(map[string]any, len(a.state))
		for k, v := range a.state {
			state[k] = v
		}
		a.mu.RUnlock()

		// Create state update message
		stateUpdatePayload, _ := json.Marshal(state)
		stateUpdate := message.New(
			uuid.New().String(),
			a.deviceID,
			"twin:"+a.deviceID,
			message.TypeEvent,
			stateUpdatePayload,
		)
		stateUpdate.SetContentType("application/json")

		// Send state update to twin
		ctx.Send(a.twinPID, stateUpdate)
	}
}

// handleEventMessage handles event messages
func (a *DeviceActor) handleEventMessage(ctx actor.Context, msg *message.Message) {
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
		// Update device state
		a.mu.Lock()
		for k, v := range event.Payload {
			a.state[k] = v
		}
		a.lastUpdated = time.Now()
		a.mu.Unlock()

		// If we have a twin, update it with the new state
		if a.twinPID != nil {
			a.mu.RLock()
			state := make(map[string]any, len(a.state))
			for k, v := range a.state {
				state[k] = v
			}
			a.mu.RUnlock()

			// Create state update message
			stateUpdatePayload, _ := json.Marshal(state)
			stateUpdate := message.New(
				uuid.New().String(),
				a.deviceID,
				"twin:"+a.deviceID,
				message.TypeEvent,
				stateUpdatePayload,
			)
			stateUpdate.SetContentType("application/json")

			// Send state update to twin
			ctx.Send(a.twinPID, stateUpdate)
		}

	case "alert":
		// Handle alert event
		a.logger.Info("Device alert", "device_id", a.deviceID, "alert", event.Payload)

		// Update device state with alert information
		a.mu.Lock()
		a.state["last_alert"] = event.Payload
		a.state["last_alert_time"] = time.Now().Format(time.RFC3339)
		a.mu.Unlock()

	default:
		a.logger.Warn("Unknown event type", "type", event.Type)
	}
}

// handleTelemetryMessage handles telemetry messages
func (a *DeviceActor) handleTelemetryMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling telemetry message", "message_id", msg.ID)

	// Parse telemetry data
	var telemetry map[string]any
	if err := json.Unmarshal(msg.Payload, &telemetry); err != nil {
		a.logger.Error("Failed to parse telemetry payload", "error", err)
		return
	}

	// Update device state with telemetry data
	a.mu.Lock()
	for k, v := range telemetry {
		a.state[k] = v
	}
	a.lastUpdated = time.Now()
	a.mu.Unlock()

	// If we have a twin, update it with the new telemetry data
	if a.twinPID != nil {
		// Create state update message
		stateUpdate := message.New(
			uuid.New().String(),
			a.deviceID,
			"twin:"+a.deviceID,
			message.TypeTelemetry,
			msg.Payload,
		)
		stateUpdate.SetContentType(msg.ContentType)

		// Send telemetry to twin
		ctx.Send(a.twinPID, stateUpdate)
	}

	// Check for threshold violations
	a.checkThresholds(ctx, telemetry)
}

// sendErrorResponse sends an error response to the original sender
func (a *DeviceActor) sendErrorResponse(ctx actor.Context, originalMsg *message.Message, code string, msg string) {
	payload, _ := json.Marshal(map[string]any{
		"status":  "error",
		"code":    code,
		"message": msg,
	})

	response := message.New(
		uuid.New().String(),
		a.deviceID,
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

// checkThresholds checks telemetry values against defined thresholds
func (a *DeviceActor) checkThresholds(ctx actor.Context, telemetry map[string]any) {
	violations := make(map[string]any)

	a.mu.RLock()
	for key, threshold := range a.thresholds {
		if value, ok := telemetry[key]; ok {
			// Try to convert value to float64
			var floatValue float64
			switch v := value.(type) {
			case float64:
				floatValue = v
			case float32:
				floatValue = float64(v)
			case int:
				floatValue = float64(v)
			case int64:
				floatValue = float64(v)
			case int32:
				floatValue = float64(v)
			default:
				continue // Skip if not a numeric type
			}

			// Check if value is outside threshold
			if floatValue < threshold.Min || floatValue > threshold.Max {
				violations[key] = map[string]any{
					"value":     floatValue,
					"min":       threshold.Min,
					"max":       threshold.Max,
					"timestamp": time.Now().Format(time.RFC3339),
				}
			}
		}
	}
	a.mu.RUnlock()

	// If there are violations and alerting is enabled, send an alert
	if len(violations) > 0 {
		a.logger.Warn("Threshold violations detected", "device_id", a.deviceID, "violations", violations)

		// Update device state
		a.mu.Lock()
		a.state["threshold_violations"] = violations
		a.state["last_violation_time"] = time.Now().Format(time.RFC3339)
		a.mu.Unlock()

		// Create alert event
		alertPayload, _ := json.Marshal(map[string]any{
			"type": "threshold_violation",
			"payload": map[string]any{
				"device_id":  a.deviceID,
				"violations": violations,
				"timestamp":  time.Now().Format(time.RFC3339),
			},
		})

		alert := message.New(
			uuid.New().String(),
			a.deviceID,
			"alerts",
			message.TypeEvent,
			alertPayload,
		)
		alert.SetContentType("application/json")

		// In a real implementation, we would send this to an alert manager
		// For now, we'll just log it
		a.logger.Info("Alert generated", "alert", alert)
	}
}
