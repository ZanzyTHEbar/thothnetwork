package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/adapters"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Config holds configuration for the WebSocket adapter
type Config struct {
	Host            string
	Port            int
	Path            string
	ReadBufferSize  int
	WriteBufferSize int
	PingInterval    time.Duration
	PongWait        time.Duration
}

// Connection represents a WebSocket connection
type Connection struct {
	ID         string
	DeviceID   string
	Conn       *websocket.Conn
	Send       chan []byte
	LastActive time.Time
}

// Adapter is a WebSocket implementation of the ProtocolAdapter interface
type Adapter struct {
	config      Config
	server      *http.Server
	upgrader    websocket.Upgrader
	connections map[string]*Connection
	deviceConns map[string]*Connection
	handler     adapters.MessageHandler
	logger      logger.Logger
	status      adapters.AdapterStatus
	mu          sync.RWMutex
}

// NewAdapter creates a new WebSocket adapter
func NewAdapter(config Config, logger logger.Logger) *Adapter {
	return &Adapter{
		config: config,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins
			},
		},
		connections: make(map[string]*Connection),
		deviceConns: make(map[string]*Connection),
		logger:      logger.With("component", "websocket_adapter"),
		status:      adapters.StatusStopped,
	}
}

// Start starts the adapter
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if already started
	if a.status != adapters.StatusStopped {
		builder := errbuilder.NewErrBuilder().
			WithMsg("Adapter already started").
			WithCode(errbuilder.CodeFailedPrecondition)
		return builder
	}

	a.status = adapters.StatusStarting

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc(a.config.Path, a.handleWebSocket)

	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.config.Host, a.config.Port),
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		a.logger.Info("Starting WebSocket server", "addr", a.server.Addr, "path", a.config.Path)

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("WebSocket server error", "error", err)
			a.mu.Lock()
			a.status = adapters.StatusError
			a.mu.Unlock()
		}
	}()

	// Start connection cleanup goroutine
	go a.cleanupConnections(ctx)

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

	// Close all connections
	for _, conn := range a.connections {
		if err := conn.Conn.Close(); err != nil {
			a.logger.Warn("Failed to close connection", "conn_id", conn.ID, "error", err)
		}
	}

	// Shutdown server
	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			builder := errbuilder.NewErrBuilder().
				WithMsg(fmt.Sprintf("Failed to shutdown WebSocket server on %s:%d", a.config.Host, a.config.Port)).
				WithCause(err).
				WithCode(errbuilder.CodeInternal)
			return builder
		}
	}

	a.connections = make(map[string]*Connection)
	a.deviceConns = make(map[string]*Connection)
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
	a.mu.RLock()
	conn, exists := a.deviceConns[deviceID]
	a.mu.RUnlock()

	if !exists {
		builder := errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Device not connected: %s", deviceID)).
			WithCode(errbuilder.CodeNotFound)
		return builder
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		builder := errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to marshal message ID: %s, Type: %s", msg.ID, msg.Type)).
			WithCause(err).
			WithCode(errbuilder.CodeInternal)
		return builder
	}

	// Send message
	select {
	case conn.Send <- data:
		return nil
	default:
		// Channel full, close connection
		a.closeConnection(conn)
		builder := errbuilder.NewErrBuilder().
			WithMsg(fmt.Sprintf("Failed to send message, connection buffer full for device: %s, buffer size: %d", deviceID, len(conn.Send))).
			WithCode(errbuilder.CodeResourceExhausted)
		return builder
	}
}

// handleWebSocket handles WebSocket connections
func (a *Adapter) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	wsConn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		a.logger.Error("Failed to upgrade connection", "error", err)
		return
	}

	// Get device ID from query parameters
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		a.logger.Warn("Connection attempt without device ID")
		wsConn.Close()
		return
	}

	// Create connection
	conn := &Connection{
		ID:         uuid.New().String(),
		DeviceID:   deviceID,
		Conn:       wsConn,
		Send:       make(chan []byte, 256),
		LastActive: time.Now(),
	}

	// Register connection
	a.mu.Lock()
	a.connections[conn.ID] = conn
	a.deviceConns[deviceID] = conn
	a.mu.Unlock()

	a.logger.Info("Device connected", "device_id", deviceID, "conn_id", conn.ID)

	// Start goroutines for reading and writing
	go a.readPump(conn)
	go a.writePump(conn)
}

// readPump pumps messages from the WebSocket connection to the handler
func (a *Adapter) readPump(conn *Connection) {
	defer func() {
		a.closeConnection(conn)
	}()

	// Set read deadline
	conn.Conn.SetReadDeadline(time.Now().Add(a.config.PongWait))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(a.config.PongWait))
		conn.LastActive = time.Now()
		return nil
	})

	for {
		_, data, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				a.logger.Warn("WebSocket error", "error", err)
			}
			break
		}

		// Update last active time
		conn.LastActive = time.Now()

		// Parse message
		var msg message.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			a.logger.Error("Failed to unmarshal message", "error", err)
			continue
		}

		// Set source to device ID if not set
		if msg.Source == "" {
			msg.Source = conn.DeviceID
		}

		// Handle message
		a.mu.RLock()
		handler := a.handler
		a.mu.RUnlock()

		if handler != nil {
			if err := handler(context.Background(), &msg); err != nil {
				a.logger.Error("Failed to handle message", "error", err)
			}
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (a *Adapter) writePump(conn *Connection) {
	ticker := time.NewTicker(a.config.PingInterval)
	defer func() {
		ticker.Stop()
		a.closeConnection(conn)
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(conn.Send)
			for range make([]struct{}, n) {
				w.Write([]byte{'\n'})
				w.Write(<-conn.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// closeConnection closes a WebSocket connection
func (a *Adapter) closeConnection(conn *Connection) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if connection is already closed
	if _, exists := a.connections[conn.ID]; !exists {
		return
	}

	// Close connection
	if err := conn.Conn.Close(); err != nil {
		a.logger.Warn("Error closing connection", "error", err)
	}

	// Remove from maps
	delete(a.connections, conn.ID)
	delete(a.deviceConns, conn.DeviceID)

	// Close send channel
	close(conn.Send)

	a.logger.Info("Device disconnected", "device_id", conn.DeviceID, "conn_id", conn.ID)
}

// cleanupConnections periodically cleans up inactive connections
func (a *Adapter) cleanupConnections(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.mu.Lock()
			now := time.Now()
			for _, conn := range a.connections {
				// Close connections that haven't been active for twice the pong wait
				if now.Sub(conn.LastActive) > 2*a.config.PongWait {
					a.logger.Info("Closing inactive connection", "device_id", conn.DeviceID, "conn_id", conn.ID)
					conn.Conn.Close()
					delete(a.connections, conn.ID)
					delete(a.deviceConns, conn.DeviceID)
					close(conn.Send)
				}
			}
			a.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
