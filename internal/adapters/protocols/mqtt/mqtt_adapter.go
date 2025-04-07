package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/adapters"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Config holds configuration for the MQTT adapter
type Config struct {
	BrokerURL          string
	ClientID           string
	Username           string
	Password           string
	CleanSession       bool
	QoS                byte
	ConnectTimeout     time.Duration
	KeepAlive          time.Duration
	PingTimeout        time.Duration
	ConnectRetryDelay  time.Duration
	MaxReconnectAttempts int
	TopicPrefix        string
}

// Adapter is an MQTT implementation of the ProtocolAdapter interface
type Adapter struct {
	config      Config
	client      mqtt.Client
	handler     adapters.MessageHandler
	logger      logger.Logger
	status      adapters.AdapterStatus
	deviceTopics map[string]string
	mu          sync.RWMutex
}

// NewAdapter creates a new MQTT adapter
func NewAdapter(config Config, logger logger.Logger) *Adapter {
	return &Adapter{
		config:       config,
		logger:       logger.With("component", "mqtt_adapter"),
		status:       adapters.StatusStopped,
		deviceTopics: make(map[string]string),
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

	// Create MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(a.config.BrokerURL).
		SetClientID(a.config.ClientID).
		SetUsername(a.config.Username).
		SetPassword(a.config.Password).
		SetCleanSession(a.config.CleanSession).
		SetConnectTimeout(a.config.ConnectTimeout).
		SetKeepAlive(a.config.KeepAlive).
		SetPingTimeout(a.config.PingTimeout).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(a.config.ConnectRetryDelay).
		SetConnectionLostHandler(a.onConnectionLost).
		SetOnConnectHandler(a.onConnect)

	// Create client
	a.client = mqtt.NewClient(opts)

	// Connect to broker
	token := a.client.Connect()
	if !token.WaitTimeout(a.config.ConnectTimeout) {
		return errbuilder.GenericErr("Connection timeout", nil)
	}
	if err := token.Error(); err != nil {
		a.status = adapters.StatusError
		return errbuilder.GenericErr("Failed to connect to MQTT broker", err)
	}

	a.status = adapters.StatusRunning
	a.logger.Info("Connected to MQTT broker", "broker", a.config.BrokerURL)

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

	// Disconnect client
	if a.client != nil && a.client.IsConnected() {
		a.client.Disconnect(250)
	}

	a.status = adapters.StatusStopped
	a.logger.Info("Disconnected from MQTT broker")

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
	defer a.mu.RUnlock()

	// Check if connected
	if a.client == nil || !a.client.IsConnected() {
		return errbuilder.GenericErr("Not connected to MQTT broker", nil)
	}

	// Get device topic
	topic, exists := a.deviceTopics[deviceID]
	if !exists {
		// Use default topic pattern if device topic not known
		topic = fmt.Sprintf("%s/devices/%s/commands", a.config.TopicPrefix, deviceID)
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return errbuilder.GenericErr("Failed to marshal message", err)
	}

	// Publish message
	token := a.client.Publish(topic, a.config.QoS, false, data)
	if !token.WaitTimeout(5 * time.Second) {
		return errbuilder.GenericErr("Publish timeout", nil)
	}
	if err := token.Error(); err != nil {
		return errbuilder.GenericErr("Failed to publish message", err)
	}

	return nil
}

// onConnect is called when the client connects to the broker
func (a *Adapter) onConnect(client mqtt.Client) {
	a.logger.Info("Connected to MQTT broker")

	// Subscribe to device topics
	token := client.Subscribe(fmt.Sprintf("%s/devices/+/+", a.config.TopicPrefix), a.config.QoS, a.handleDeviceMessage)
	if token.Wait() && token.Error() != nil {
		a.logger.Error("Failed to subscribe to device topics", "error", token.Error())
	}

	// Subscribe to broadcast topic
	token = client.Subscribe(fmt.Sprintf("%s/broadcast", a.config.TopicPrefix), a.config.QoS, a.handleBroadcastMessage)
	if token.Wait() && token.Error() != nil {
		a.logger.Error("Failed to subscribe to broadcast topic", "error", token.Error())
	}
}

// onConnectionLost is called when the client loses connection to the broker
func (a *Adapter) onConnectionLost(client mqtt.Client, err error) {
	a.logger.Warn("Lost connection to MQTT broker", "error", err)
	a.mu.Lock()
	a.status = adapters.StatusError
	a.mu.Unlock()
}

// handleDeviceMessage handles messages from devices
func (a *Adapter) handleDeviceMessage(client mqtt.Client, mqttMsg mqtt.Message) {
	// Extract device ID and message type from topic
	// Topic format: {prefix}/devices/{device_id}/{message_type}
	topic := mqttMsg.Topic()
	deviceID, msgType, err := parseDeviceTopic(topic, a.config.TopicPrefix)
	if err != nil {
		a.logger.Error("Failed to parse device topic", "topic", topic, "error", err)
		return
	}

	// Store device topic for future messages
	a.mu.Lock()
	a.deviceTopics[deviceID] = topic[:len(topic)-len(msgType)-1] // Remove message type from topic
	a.mu.Unlock()

	// Parse message type
	var messageType message.Type
	switch msgType {
	case "telemetry":
		messageType = message.TypeTelemetry
	case "events":
		messageType = message.TypeEvent
	case "responses":
		messageType = message.TypeResponse
	default:
		a.logger.Warn("Unknown message type", "type", msgType)
		return
	}

	// Create message
	msg := message.NewMessage(
		uuid.New().String(),
		deviceID,
		"*", // Target will be determined by the handler
		messageType,
		mqttMsg.Payload(),
	)
	msg.SetContentType("application/json")

	// Handle message
	a.mu.RLock()
	handler := a.handler
	a.mu.RUnlock()

	if handler != nil {
		if err := handler(context.Background(), msg); err != nil {
			a.logger.Error("Failed to handle message", "error", err)
		}
	}
}

// handleBroadcastMessage handles broadcast messages
func (a *Adapter) handleBroadcastMessage(client mqtt.Client, mqttMsg mqtt.Message) {
	// Create message
	msg := message.NewMessage(
		uuid.New().String(),
		"broadcast",
		"*", // Target will be determined by the handler
		message.TypeCommand,
		mqttMsg.Payload(),
	)
	msg.SetContentType("application/json")

	// Handle message
	a.mu.RLock()
	handler := a.handler
	a.mu.RUnlock()

	if handler != nil {
		if err := handler(context.Background(), msg); err != nil {
			a.logger.Error("Failed to handle message", "error", err)
		}
	}
}

// parseDeviceTopic parses a device topic to extract device ID and message type
// Topic format: {prefix}/devices/{device_id}/{message_type}
func parseDeviceTopic(topic, prefix string) (deviceID, msgType string, err error) {
	// Check if topic has the correct prefix
	if len(topic) <= len(prefix)+9 { // +9 for "/devices/"
		return "", "", errbuilder.GenericErr("Invalid topic format", nil)
	}

	// Split topic
	parts := make([]string, 0)
	start := len(prefix) + 1 // +1 for "/"
	if prefix != "" {
		if topic[:start-1] != prefix {
			return "", "", errbuilder.GenericErr("Invalid topic prefix", nil)
		}
		parts = append(parts, prefix)
	} else {
		start = 0
	}

	// Extract parts
	current := start
	for i := start; i < len(topic); i++ {
		if topic[i] == '/' {
			parts = append(parts, topic[current:i])
			current = i + 1
		}
	}
	parts = append(parts, topic[current:])

	// Check if we have enough parts
	if len(parts) < 3 {
		return "", "", errbuilder.GenericErr("Invalid topic format", nil)
	}

	// Check if second part is "devices"
	if parts[1] != "devices" {
		return "", "", errbuilder.GenericErr("Invalid topic format", nil)
	}

	// Extract device ID and message type
	deviceID = parts[2]
	msgType = parts[3]

	return deviceID, msgType, nil
}
