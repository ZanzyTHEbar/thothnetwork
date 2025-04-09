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

// RoomActor represents a room actor
type RoomActor struct {
	roomID      string
	devicePIDs  map[string]*actor.PID
	state       map[string]any
	lastUpdated time.Time
	logger      logger.Logger
	mu          sync.RWMutex
}

// NewRoomActor creates a new room actor
func NewRoomActor(roomID string, logger logger.Logger) *RoomActor {
	return &RoomActor{
		roomID:      roomID,
		devicePIDs:  make(map[string]*actor.PID),
		state:       make(map[string]any),
		lastUpdated: time.Now(),
		logger:      logger.With("actor", "room", "room_id", roomID),
	}
}

// Receive handles incoming messages
func (a *RoomActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *message.Message:
		a.handleMessage(ctx, msg)
	case *AddDeviceCommand:
		a.handleAddDevice(ctx, msg)
	case *RemoveDeviceCommand:
		a.handleRemoveDevice(ctx, msg)
	case *GetDevicesQuery:
		a.handleGetDevices(ctx, msg)
	default:
		a.logger.Debug("Unknown message type", "type", ctx.Message())
	}
}

// handleMessage handles incoming messages
func (a *RoomActor) handleMessage(ctx actor.Context, msg *message.Message) {
	a.logger.Debug("Handling message", "message_id", msg.ID, "type", msg.Type)

	switch msg.Type {
	case message.TypeCommand:
		// Handle command message
		a.handleCommandMessage(ctx, msg)
	case message.TypeEvent:
		// Handle event message
		a.handleEventMessage(ctx, msg)
	default:
		a.logger.Warn("Unknown message type", "type", msg.Type)
	}
}

// handleCommandMessage handles command messages
func (a *RoomActor) handleCommandMessage(ctx actor.Context, msg *message.Message) {
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
	case "broadcast":
		// Broadcast command to all devices in the room
		if payload, ok := command.Params["payload"]; ok {
			a.broadcastToDevices(ctx, payload)
		} else {
			a.sendErrorResponse(ctx, msg, "invalid_params", "Missing payload parameter")
			return
		}

		// Create response payload
		responsePayload, err = json.Marshal(map[string]any{
			"status":  "success",
			"message": "Command broadcast to all devices",
		})

	case "get_state":
		// Read room state
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

	case "set_state":
		// Update room state
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
		a.roomID,
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

// handleEventMessage handles event messages
func (a *RoomActor) handleEventMessage(ctx actor.Context, msg *message.Message) {
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
	case "device_state_changed":
		// Update room state with device state
		if deviceID, ok := event.Payload["device_id"].(string); ok {
			a.mu.Lock()
			if a.state["devices"] == nil {
				a.state["devices"] = make(map[string]any)
			}
			devices, _ := a.state["devices"].(map[string]any)
			devices[deviceID] = event.Payload["state"]
			a.lastUpdated = time.Now()
			a.mu.Unlock()
		}

	case "occupancy_changed":
		// Update room occupancy
		if occupancy, ok := event.Payload["occupancy"].(float64); ok {
			a.mu.Lock()
			a.state["occupancy"] = occupancy
			a.lastUpdated = time.Now()
			a.mu.Unlock()
		}

	default:
		a.logger.Warn("Unknown event type", "type", event.Type)
	}
}

// handleAddDevice handles adding a device to the room
func (a *RoomActor) handleAddDevice(ctx actor.Context, cmd *AddDeviceCommand) {
	a.logger.Debug("Adding device to room", "device_id", cmd.DeviceID)

	// Add device to the room
	a.mu.Lock()
	a.devicePIDs[cmd.DeviceID] = cmd.DevicePID
	a.mu.Unlock()

	// Send response
	ctx.Respond(&AddDeviceResponse{Success: true})
}

// handleRemoveDevice handles removing a device from the room
func (a *RoomActor) handleRemoveDevice(ctx actor.Context, cmd *RemoveDeviceCommand) {
	a.logger.Debug("Removing device from room", "device_id", cmd.DeviceID)

	// Remove device from the room
	a.mu.Lock()
	delete(a.devicePIDs, cmd.DeviceID)
	a.mu.Unlock()

	// Send response
	ctx.Respond(&RemoveDeviceResponse{Success: true})
}

// handleGetDevices handles getting all devices in the room
func (a *RoomActor) handleGetDevices(ctx actor.Context, _ *GetDevicesQuery) {
	a.logger.Debug("Getting devices in room")

	// Get devices
	a.mu.RLock()
	devices := make(map[string]*actor.PID, len(a.devicePIDs))
	for k, v := range a.devicePIDs {
		devices[k] = v
	}
	a.mu.RUnlock()

	// Send response
	ctx.Respond(&GetDevicesResponse{Devices: devices})
}

// broadcastToDevices broadcasts a message to all devices in the room
func (a *RoomActor) broadcastToDevices(ctx actor.Context, payload any) {
	// Get devices
	a.mu.RLock()
	devices := make(map[string]*actor.PID, len(a.devicePIDs))
	for k, v := range a.devicePIDs {
		devices[k] = v
	}
	a.mu.RUnlock()

	// Create command payload
	commandPayload, _ := json.Marshal(payload)

	// Send command to each device
	for deviceID, pid := range devices {
		command := message.NewMessage(
			uuid.New().String(),
			a.roomID,
			deviceID,
			message.TypeCommand,
			commandPayload,
		)
		command.SetContentType("application/json")

		ctx.Send(pid, command)
	}
}

// sendErrorResponse sends an error response to the original sender
func (a *RoomActor) sendErrorResponse(ctx actor.Context, originalMsg *message.Message, code string, message string) {
	payload, _ := json.Marshal(map[string]any{
		"status":  "error",
		"code":    code,
		"message": message,
	})

	response := message.NewMessage(
		uuid.New().String(),
		a.roomID,
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

// AddDeviceCommand is a command to add a device to a room
type AddDeviceCommand struct {
	DeviceID  string
	DevicePID *actor.PID
}

// AddDeviceResponse is a response to an AddDeviceCommand
type AddDeviceResponse struct {
	Success bool
}

// RemoveDeviceCommand is a command to remove a device from a room
type RemoveDeviceCommand struct {
	DeviceID string
}

// RemoveDeviceResponse is a response to a RemoveDeviceCommand
type RemoveDeviceResponse struct {
	Success bool
}

// GetDevicesQuery is a query to get all devices in a room
type GetDevicesQuery struct{}

// GetDevicesResponse is a response to a GetDevicesQuery
type GetDevicesResponse struct {
	Devices map[string]*actor.PID
}
