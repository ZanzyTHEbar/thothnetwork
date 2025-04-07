package brokers

import (
	"context"
	"time"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
)

// MessageHandler is a function that handles messages
type MessageHandler func(ctx context.Context, msg *message.Message) error

// ConsumerConfig defines the configuration for a consumer
type ConsumerConfig struct {
	// Name is the name of the consumer
	Name string

	// DeliverPolicy defines the delivery policy for the consumer
	DeliverPolicy string

	// AckPolicy defines the acknowledgement policy for the consumer
	AckPolicy string

	// FilterSubject defines a subject filter for the consumer
	FilterSubject string

	// MaxDeliver defines the maximum number of delivery attempts
	MaxDeliver int

	// AckWait defines the acknowledgement wait time
	AckWait time.Duration
}

// Subscription represents a subscription to a topic
type Subscription interface {
	// Unsubscribe unsubscribes from the topic
	Unsubscribe() error

	// Topic returns the topic of the subscription
	Topic() string
}

// MessageBroker defines the interface for message brokers
type MessageBroker interface {
	// Connect connects to the message broker
	Connect(ctx context.Context) error

	// Disconnect disconnects from the message broker
	Disconnect(ctx context.Context) error

	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, msg *message.Message) error

	// Subscribe subscribes to a topic
	Subscribe(ctx context.Context, topic string, handler MessageHandler) (Subscription, error)

	// Request sends a request message and waits for a response
	Request(ctx context.Context, topic string, msg *message.Message, timeout time.Duration) (*message.Message, error)

	// CreateStream creates a new stream
	CreateStream(ctx context.Context, name string, subjects []string) error

	// DeleteStream deletes a stream
	DeleteStream(ctx context.Context, name string) error

	// CreateConsumer creates a new consumer for a stream
	CreateConsumer(ctx context.Context, stream string, config ConsumerConfig) error

	// DeleteConsumer deletes a consumer
	DeleteConsumer(ctx context.Context, stream string, consumer string) error

	// PublishToStream publishes a message to a stream
	PublishToStream(ctx context.Context, stream string, msg *message.Message) error

	// SubscribeToStream subscribes to a stream
	SubscribeToStream(ctx context.Context, stream string, consumer string, handler MessageHandler) (Subscription, error)
}
