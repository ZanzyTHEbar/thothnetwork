package nats

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/ports/brokers"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Subscription implements the brokers.Subscription interface
type Subscription struct {
	sub    *nats.Subscription
	topic  string
	broker *MessageBroker
}

// Unsubscribe unsubscribes from the topic
func (s *Subscription) Unsubscribe() error {
	return s.sub.Unsubscribe()
}

// Topic returns the topic of the subscription
func (s *Subscription) Topic() string {
	return s.topic
}

// MessageBroker is a NATS implementation of the MessageBroker interface
type MessageBroker struct {
	conn          *nats.Conn
	js            jetstream.JetStream
	config        Config
	logger        logger.Logger
	mu            sync.RWMutex
	subscriptions map[string][]*Subscription
	consumers     map[string]map[string]jetstream.Consumer
}

// Config holds configuration for the NATS message broker
type Config struct {
	URL      string
	Username string
	Password string
	Token    string
}

// NewMessageBroker creates a new NATS message broker
func NewMessageBroker(config Config, logger logger.Logger) *MessageBroker {
	return &MessageBroker{
		config:        config,
		logger:        logger.With("component", "nats_broker"),
		subscriptions: make(map[string][]*Subscription),
		consumers:     make(map[string]map[string]jetstream.Consumer),
	}
}

// Connect connects to the NATS server
func (b *MessageBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if already connected
	if b.conn != nil {
		return nil
	}

	// Create connection options
	opts := []nats.Option{
		nats.Name("thothnetwork"),
		nats.ReconnectWait(nats.DefaultReconnectWait),
		nats.MaxReconnects(nats.DefaultMaxReconnect),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			b.logger.Warn("NATS disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			b.logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			b.logger.Info("NATS connection closed")
		}),
	}

	// Add authentication options
	if b.config.Username != "" && b.config.Password != "" {
		opts = append(opts, nats.UserInfo(b.config.Username, b.config.Password))
	} else if b.config.Token != "" {
		opts = append(opts, nats.Token(b.config.Token))
	}

	// Connect to NATS
	conn, err := nats.Connect(b.config.URL, opts...)
	if err != nil {
		return errbuilder.GenericErr("Failed to connect to NATS", err)
	}

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return errbuilder.GenericErr("Failed to create JetStream context", err)
	}

	b.conn = conn
	b.js = js

	b.logger.Info("Connected to NATS", "url", conn.ConnectedUrl())
	return nil
}

// Disconnect disconnects from the NATS server
func (b *MessageBroker) Disconnect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if connected
	if b.conn == nil {
		return nil
	}

	// Unsubscribe from all subscriptions
	for _, subs := range b.subscriptions {
		for _, sub := range subs {
			if err := sub.Unsubscribe(); err != nil {
				b.logger.Warn("Failed to unsubscribe", "topic", sub.Topic(), "error", err)
			}
		}
	}

	// Close connection
	b.conn.Close()
	b.conn = nil
	b.js = nil
	b.subscriptions = make(map[string][]*Subscription)

	b.logger.Info("Disconnected from NATS")
	return nil
}

// Publish publishes a message to a topic
func (b *MessageBroker) Publish(ctx context.Context, topic string, msg *message.Message) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check if connected
	if b.conn == nil {
		return errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return errbuilder.GenericErr("Failed to marshal message", err)
	}

	// Publish message
	if err := b.conn.Publish(topic, data); err != nil {
		return errbuilder.GenericErr("Failed to publish message", err)
	}

	return nil
}

// Subscribe subscribes to a topic
func (b *MessageBroker) Subscribe(ctx context.Context, topic string, handler brokers.MessageHandler) (brokers.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if connected
	if b.conn == nil {
		return nil, errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Create subscription
	sub, err := b.conn.Subscribe(topic, func(m *nats.Msg) {
		// Unmarshal message
		var msg message.Message
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			b.logger.Error("Failed to unmarshal message", "error", err)
			return
		}

		// Handle message
		if err := handler(ctx, &msg); err != nil {
			b.logger.Error("Failed to handle message", "error", err)
		}
	})

	if err != nil {
		return nil, errbuilder.GenericErr("Failed to subscribe to topic", err)
	}

	// Create subscription object
	subscription := &Subscription{
		sub:    sub,
		topic:  topic,
		broker: b,
	}

	// Add to subscriptions map
	b.subscriptions[topic] = append(b.subscriptions[topic], subscription)

	return subscription, nil
}

// CreateStream creates a new stream
func (b *MessageBroker) CreateStream(ctx context.Context, name string, subjects []string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check if connected
	if b.conn == nil || b.js == nil {
		return errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Create stream configuration
	cfg := jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
	}

	// Create stream
	_, err := b.js.CreateStream(ctx, cfg)
	if err != nil {
		return errbuilder.GenericErr("Failed to create stream", err)
	}

	return nil
}

// DeleteStream deletes a stream
func (b *MessageBroker) DeleteStream(ctx context.Context, name string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check if connected
	if b.conn == nil || b.js == nil {
		return errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Delete stream
	if err := b.js.DeleteStream(ctx, name); err != nil {
		return errbuilder.GenericErr("Failed to delete stream", err)
	}

	return nil
}

// Request sends a request message and waits for a response
func (b *MessageBroker) Request(ctx context.Context, topic string, msg *message.Message, timeout time.Duration) (*message.Message, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check if connected
	if b.conn == nil {
		return nil, errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to marshal message", err)
	}

	// Send request
	resp, err := b.conn.Request(topic, data, timeout)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to send request", err)
	}

	// Unmarshal response
	var respMsg message.Message
	if err := json.Unmarshal(resp.Data, &respMsg); err != nil {
		return nil, errbuilder.GenericErr("Failed to unmarshal response", err)
	}

	return &respMsg, nil
}

// CreateConsumer creates a new consumer for a stream
func (b *MessageBroker) CreateConsumer(ctx context.Context, stream string, config brokers.ConsumerConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if connected
	if b.conn == nil || b.js == nil {
		return errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Get stream
	str, err := b.js.Stream(ctx, stream)
	if err != nil {
		return errbuilder.GenericErr("Failed to get stream", err)
	}

	// Create consumer configuration
	cfg := jetstream.ConsumerConfig{
		Name:          config.Name,
		FilterSubject: config.FilterSubject,
		MaxDeliver:    config.MaxDeliver,
		AckWait:       config.AckWait,
	}

	// Set deliver policy
	switch config.DeliverPolicy {
	case "all":
		cfg.DeliverPolicy = jetstream.DeliverAllPolicy
	case "last":
		cfg.DeliverPolicy = jetstream.DeliverLastPolicy
	case "new":
		cfg.DeliverPolicy = jetstream.DeliverNewPolicy
	default:
		cfg.DeliverPolicy = jetstream.DeliverAllPolicy
	}

	// Set ack policy
	switch config.AckPolicy {
	case "explicit":
		cfg.AckPolicy = jetstream.AckExplicitPolicy
	case "all":
		cfg.AckPolicy = jetstream.AckAllPolicy
	case "none":
		cfg.AckPolicy = jetstream.AckNonePolicy
	default:
		cfg.AckPolicy = jetstream.AckExplicitPolicy
	}

	// Create consumer
	consumer, err := str.CreateOrUpdateConsumer(ctx, cfg)
	if err != nil {
		return errbuilder.GenericErr("Failed to create consumer", err)
	}

	// Store consumer
	if _, exists := b.consumers[stream]; !exists {
		b.consumers[stream] = make(map[string]jetstream.Consumer)
	}
	b.consumers[stream][config.Name] = consumer

	return nil
}

// DeleteConsumer deletes a consumer
func (b *MessageBroker) DeleteConsumer(ctx context.Context, stream string, consumer string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if connected
	if b.conn == nil || b.js == nil {
		return errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Get stream
	str, err := b.js.Stream(ctx, stream)
	if err != nil {
		return errbuilder.GenericErr("Failed to get stream", err)
	}

	// Delete consumer
	if err := str.DeleteConsumer(ctx, consumer); err != nil {
		return errbuilder.GenericErr("Failed to delete consumer", err)
	}

	// Remove consumer from map
	if streamConsumers, exists := b.consumers[stream]; exists {
		delete(streamConsumers, consumer)
	}

	return nil
}

// PublishToStream publishes a message to a stream
func (b *MessageBroker) PublishToStream(ctx context.Context, stream string, msg *message.Message) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check if connected
	if b.conn == nil || b.js == nil {
		return errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return errbuilder.GenericErr("Failed to marshal message", err)
	}

	// Check if stream exists
	_, err = b.js.Stream(ctx, stream)
	if err != nil {
		return errbuilder.GenericErr("Stream not found", err)
	}

	// Publish message
	_, err = b.js.Publish(ctx, stream+"."+msg.ID, data)
	if err != nil {
		return errbuilder.GenericErr("Failed to publish message to stream", err)
	}

	return nil
}

// SubscribeToStream subscribes to a stream
func (b *MessageBroker) SubscribeToStream(ctx context.Context, stream string, consumer string, handler brokers.MessageHandler) (brokers.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if connected
	if b.conn == nil || b.js == nil {
		return nil, errbuilder.GenericErr("Not connected to NATS", nil)
	}

	// Get consumer
	consumerObj, exists := b.consumers[stream][consumer]
	if !exists {
		return nil, errbuilder.GenericErr("Consumer not found", nil)
	}

	// Create message handler
	msgHandler := func(msg jetstream.Msg) {
		// Unmarshal message
		var m message.Message
		if err := json.Unmarshal(msg.Data(), &m); err != nil {
			b.logger.Error("Failed to unmarshal message", "error", err)
			return
		}

		// Handle message
		if err := handler(ctx, &m); err != nil {
			b.logger.Error("Failed to handle message", "error", err)
			return
		}

		// Acknowledge message
		if err := msg.Ack(); err != nil {
			b.logger.Error("Failed to acknowledge message", "error", err)
		}
	}

	// Create consumer context
	consumerCtx, err := consumerObj.Consume(msgHandler)
	if err != nil {
		return nil, errbuilder.GenericErr("Failed to consume messages", err)
	}

	// Create subscription object
	subscription := &Subscription{
		sub:    nil,
		topic:  stream + "." + consumer,
		broker: b,
	}

	// Add to subscriptions map
	b.subscriptions[subscription.Topic()] = append(b.subscriptions[subscription.Topic()], subscription)

	// Start consumer in a goroutine
	go func() {
		<-ctx.Done()
		consumerCtx.Stop()
	}()

	return subscription, nil
}
