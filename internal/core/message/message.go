package message

import (
	"time"
)

// Type represents the type of a message
type Type string

const (
	// TypeTelemetry is for device telemetry data
	TypeTelemetry Type = "telemetry"
	// TypeCommand is for commands sent to devices
	TypeCommand Type = "command"
	// TypeEvent is for device events
	TypeEvent Type = "event"
	// TypeResponse is for responses to commands
	TypeResponse Type = "response"
)

// Message represents a message in the system
type Message struct {
	// ID is the unique identifier for the message
	ID string `json:"id"`
	
	// Source is the source of the message (usually a device ID)
	Source string `json:"source"`
	
	// Target is the target of the message (device ID, room ID, or "*" for broadcast)
	Target string `json:"target"`
	
	// Type is the type of the message
	Type Type `json:"type"`
	
	// Payload is the message payload
	Payload []byte `json:"payload"`
	
	// ContentType is the MIME type of the payload
	ContentType string `json:"content_type"`
	
	// Timestamp is when the message was created
	Timestamp time.Time `json:"timestamp"`
	
	// Metadata is additional information about the message
	Metadata map[string]string `json:"metadata"`
}

// NewMessage creates a new message
func NewMessage(id, source, target string, msgType Type, payload []byte) *Message {
	return &Message{
		ID:          id,
		Source:      source,
		Target:      target,
		Type:        msgType,
		Payload:     payload,
		ContentType: "application/octet-stream",
		Timestamp:   time.Now(),
		Metadata:    make(map[string]string),
	}
}

// SetContentType sets the content type of the message
func (m *Message) SetContentType(contentType string) {
	m.ContentType = contentType
}

// SetMetadata sets a metadata value for the message
func (m *Message) SetMetadata(key, value string) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[key] = value
}
