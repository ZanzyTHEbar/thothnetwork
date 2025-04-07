package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/adapters"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Config holds configuration for the HTTP adapter
type Config struct {
	Host            string
	Port            int
	BasePath        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxHeaderBytes  int
	ShutdownTimeout time.Duration
}

// Adapter is an HTTP implementation of the ProtocolAdapter interface
type Adapter struct {
	config  Config
	server  *http.Server
	handler adapters.MessageHandler
	logger  logger.Logger
	status  adapters.AdapterStatus
	mu      sync.RWMutex
}

// NewAdapter creates a new HTTP adapter
func NewAdapter(config Config, logger logger.Logger) *Adapter {
	return &Adapter{
		config: config,
		logger: logger.With("component", "http_adapter"),
		status: adapters.StatusStopped,
	}
}

// Start starts the adapter
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if already started
	if a.status != adapters.StatusStopped {
		return errbuilder.GenericErr("Adapter already started", nil)
	}

	a.status = adapters.StatusStarting

	// Create HTTP server
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc(fmt.Sprintf("%s/devices", a.config.BasePath), a.handleDevices)
	mux.HandleFunc(fmt.Sprintf("%s/devices/", a.config.BasePath), a.handleDevice)
	mux.HandleFunc(fmt.Sprintf("%s/messages", a.config.BasePath), a.handleMessages)

	a.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", a.config.Host, a.config.Port),
		Handler:        mux,
		ReadTimeout:    a.config.ReadTimeout,
		WriteTimeout:   a.config.WriteTimeout,
		MaxHeaderBytes: a.config.MaxHeaderBytes,
	}

	// Start server in a goroutine
	go func() {
		a.logger.Info("Starting HTTP server", "addr", a.server.Addr)

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server error", "error", err)
			a.mu.Lock()
			a.status = adapters.StatusError
			a.mu.Unlock()
		}
	}()

	a.status = adapters.StatusRunning
	return nil
}

// Stop stops the adapter
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if already stopped
	if a.status == adapters.StatusStopped {
		return nil
	}

	// Create a context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, a.config.ShutdownTimeout)
	defer cancel()

	// Shutdown server
	if a.server != nil {
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return errbuilder.GenericErr("Failed to shutdown HTTP server", err)
		}
	}

	a.status = adapters.StatusStopped
	return nil
}

// Status returns the current status of the adapter
func (a *Adapter) Status() adapters.AdapterStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// SetMessageHandler sets the handler for incoming messages
func (a *Adapter) SetMessageHandler(handler adapters.MessageHandler) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handler = handler
	return nil
}

// SendMessage sends a message to a device
func (a *Adapter) SendMessage(ctx context.Context, deviceID string, msg *message.Message) error {
	// HTTP adapter doesn't maintain persistent connections to devices
	// This would typically be implemented using a notification service or webhook
	return errbuilder.GenericErr("HTTP adapter doesn't support direct message sending", nil)
}

// handleDevices handles requests to /devices
func (a *Adapter) handleDevices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List devices
		a.handleListDevices(w, r)
	case http.MethodPost:
		// Create device
		a.handleCreateDevice(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDevice handles requests to /devices/{id}
func (a *Adapter) handleDevice(w http.ResponseWriter, r *http.Request) {
	// Extract device ID from path
	deviceID := r.URL.Path[len(fmt.Sprintf("%s/devices/", a.config.BasePath)):]
	if deviceID == "" {
		http.Error(w, "Device ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get device
		a.handleGetDevice(w, r, deviceID)
	case http.MethodPut:
		// Update device
		a.handleUpdateDevice(w, r, deviceID)
	case http.MethodDelete:
		// Delete device
		a.handleDeleteDevice(w, r, deviceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleMessages handles requests to /messages
func (a *Adapter) handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Send message
		a.handleSendMessage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListDevices handles GET /devices
func (a *Adapter) handleListDevices(w http.ResponseWriter, r *http.Request) {
	// Create a message for listing devices
	msg := message.NewMessage(
		uuid.New().String(),
		"http",
		"*",
		message.TypeCommand,
		[]byte(`{"command":"list_devices"}`),
	)
	msg.SetContentType("application/json")

	// Process message
	a.processMessage(w, r, msg)
}

// handleCreateDevice handles POST /devices
func (a *Adapter) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create command payload
	payload, err := json.Marshal(map[string]interface{}{
		"command": "create_device",
		"data":    body,
	})
	if err != nil {
		http.Error(w, "Failed to create command", http.StatusInternalServerError)
		return
	}

	// Create a message for creating a device
	msg := message.NewMessage(
		uuid.New().String(),
		"http",
		"*",
		message.TypeCommand,
		payload,
	)
	msg.SetContentType("application/json")

	// Process message
	a.processMessage(w, r, msg)
}

// handleGetDevice handles GET /devices/{id}
func (a *Adapter) handleGetDevice(w http.ResponseWriter, r *http.Request, deviceID string) {
	// Create command payload
	payload, err := json.Marshal(map[string]interface{}{
		"command":   "get_device",
		"device_id": deviceID,
	})
	if err != nil {
		http.Error(w, "Failed to create command", http.StatusInternalServerError)
		return
	}

	// Create a message for getting a device
	msg := message.NewMessage(
		uuid.New().String(),
		"http",
		"*",
		message.TypeCommand,
		payload,
	)
	msg.SetContentType("application/json")

	// Process message
	a.processMessage(w, r, msg)
}

// handleUpdateDevice handles PUT /devices/{id}
func (a *Adapter) handleUpdateDevice(w http.ResponseWriter, r *http.Request, deviceID string) {
	// Parse request body
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Add device ID to body
	body["device_id"] = deviceID

	// Create command payload
	payload, err := json.Marshal(map[string]interface{}{
		"command": "update_device",
		"data":    body,
	})
	if err != nil {
		http.Error(w, "Failed to create command", http.StatusInternalServerError)
		return
	}

	// Create a message for updating a device
	msg := message.NewMessage(
		uuid.New().String(),
		"http",
		"*",
		message.TypeCommand,
		payload,
	)
	msg.SetContentType("application/json")

	// Process message
	a.processMessage(w, r, msg)
}

// handleDeleteDevice handles DELETE /devices/{id}
func (a *Adapter) handleDeleteDevice(w http.ResponseWriter, r *http.Request, deviceID string) {
	// Create command payload
	payload, err := json.Marshal(map[string]interface{}{
		"command":   "delete_device",
		"device_id": deviceID,
	})
	if err != nil {
		http.Error(w, "Failed to create command", http.StatusInternalServerError)
		return
	}

	// Create a message for deleting a device
	msg := message.NewMessage(
		uuid.New().String(),
		"http",
		"*",
		message.TypeCommand,
		payload,
	)
	msg.SetContentType("application/json")

	// Process message
	a.processMessage(w, r, msg)
}

// handleSendMessage handles POST /messages
func (a *Adapter) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var body struct {
		Target  string          `json:"target"`
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if body.Target == "" {
		http.Error(w, "Target is required", http.StatusBadRequest)
		return
	}
	if body.Type == "" {
		http.Error(w, "Type is required", http.StatusBadRequest)
		return
	}

	// Determine message type
	var msgType message.Type
	switch body.Type {
	case "telemetry":
		msgType = message.TypeTelemetry
	case "command":
		msgType = message.TypeCommand
	case "event":
		msgType = message.TypeEvent
	case "response":
		msgType = message.TypeResponse
	default:
		http.Error(w, "Invalid message type", http.StatusBadRequest)
		return
	}

	// Create a message
	msg := message.NewMessage(
		uuid.New().String(),
		"http",
		body.Target,
		msgType,
		body.Payload,
	)
	msg.SetContentType("application/json")

	// Process message
	a.processMessage(w, r, msg)
}

// processMessage processes a message and sends the response
func (a *Adapter) processMessage(w http.ResponseWriter, r *http.Request, msg *message.Message) {
	// Get message handler
	a.mu.RLock()
	handler := a.handler
	a.mu.RUnlock()

	if handler == nil {
		http.Error(w, "Message handler not set", http.StatusInternalServerError)
		return
	}

	// Create response channel
	respChan := make(chan *message.Message, 1)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Handle message in a goroutine
	go func() {
		// Create a new context with response channel
		ctxWithValue := context.WithValue(ctx, "response_channel", respChan)

		// Handle message
		if err := handler(ctxWithValue, msg); err != nil {
			a.logger.Error("Failed to handle message", "error", err)
			respChan <- message.NewMessage(
				uuid.New().String(),
				"system",
				msg.Source,
				message.TypeResponse,
				[]byte(`{"error":"Internal server error"}`),
			)
		}
	}()

	// Wait for response or timeout
	select {
	case resp := <-respChan:
		// Set content type
		w.Header().Set("Content-Type", resp.ContentType)

		// Write response
		w.Write(resp.Payload)
	case <-ctx.Done():
		// Timeout
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
	}
}
