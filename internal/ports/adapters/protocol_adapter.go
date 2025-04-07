package adapters

import (
	"context"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
)

// AdapterStatus represents the status of an adapter
type AdapterStatus string

const (
	// StatusStopped indicates the adapter is stopped
	StatusStopped AdapterStatus = "stopped"
	// StatusStarting indicates the adapter is starting
	StatusStarting AdapterStatus = "starting"
	// StatusRunning indicates the adapter is running
	StatusRunning AdapterStatus = "running"
	// StatusError indicates the adapter is in an error state
	StatusError AdapterStatus = "error"
)

// MessageHandler is a function that handles messages
type MessageHandler func(ctx context.Context, msg *message.Message) error

// ProtocolAdapter defines the interface for protocol adapters
type ProtocolAdapter interface {
	// Start starts the adapter
	Start(ctx context.Context) error

	// Stop stops the adapter
	Stop(ctx context.Context) error

	// Status returns the current status of the adapter
	Status() AdapterStatus

	// SetMessageHandler sets the handler for incoming messages
	SetMessageHandler(handler MessageHandler) error

	// SendMessage sends a message to a device
	SendMessage(ctx context.Context, deviceID string, msg *message.Message) error
}
