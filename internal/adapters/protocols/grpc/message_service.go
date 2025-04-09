package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	thothnetworkv1 "github.com/ZanzyTHEbar/thothnetwork/pkg/gen/proto/thothnetwork/v1"
)

// MessageService implements the MessageService gRPC service
type MessageService struct {
	thothnetworkv1.UnimplementedMessageServiceServer
	adapter *Adapter
}

// NewMessageService creates a new MessageService instance
func NewMessageService(adapter *Adapter) *MessageService {
	return &MessageService{
		adapter: adapter,
	}
}

// PublishMessage publishes a message to a topic
func (s *MessageService) PublishMessage(ctx context.Context, req *thothnetworkv1.PublishMessageRequest) (*thothnetworkv1.PublishMessageResponse, error) {
	s.adapter.logger.Debug("PublishMessage request received",
		"source", req.Source,
		"target", req.Target,
		"topic", req.Topic)

	// TODO: Implement actual message publishing
	// This would typically involve:
	// 1. Converting the gRPC message to a domain message
	// 2. Using the adapter's message handler to process the message
	// 3. Returning the result

	// For now, create a mock response
	mockMsg := &thothnetworkv1.Message{
		Id:          "mock-message-id",
		Source:      req.Source,
		Target:      req.Target,
		Type:        req.Type,
		Payload:     req.Payload,
		ContentType: req.ContentType,
		Timestamp:   timestamppb.Now(),
		Metadata:    req.Metadata,
	}

	return &thothnetworkv1.PublishMessageResponse{
		Id:      mockMsg.Id,
		Message: mockMsg,
	}, nil
}

// Subscribe subscribes to a topic
func (s *MessageService) Subscribe(req *thothnetworkv1.SubscribeRequest, stream thothnetworkv1.MessageService_SubscribeServer) error {
	s.adapter.logger.Debug("Subscribe request received", "topic", req.Topic)

	// TODO: Implement subscription logic
	// This would typically involve:
	// 1. Registering a subscription to the specified topic
	// 2. Sending messages to the client as they are received
	// 3. Handling client disconnection

	// For now, return an unimplemented error
	return status.Error(codes.Unimplemented, "method Subscribe not implemented")
}

// Unsubscribe unsubscribes from a topic
func (s *MessageService) Unsubscribe(ctx context.Context, req *thothnetworkv1.UnsubscribeRequest) (*thothnetworkv1.UnsubscribeResponse, error) {
	s.adapter.logger.Debug("Unsubscribe request received", "topic", req.Topic)

	// TODO: Implement unsubscription logic

	return nil, status.Error(codes.Unimplemented, "method Unsubscribe not implemented")
}

// CreateStream creates a new stream
func (s *MessageService) CreateStream(ctx context.Context, req *thothnetworkv1.CreateStreamRequest) (*thothnetworkv1.CreateStreamResponse, error) {
	s.adapter.logger.Debug("CreateStream request received", "name", req.Name)

	// TODO: Implement stream creation logic

	return nil, status.Error(codes.Unimplemented, "method CreateStream not implemented")
}

// DeleteStream deletes a stream
func (s *MessageService) DeleteStream(ctx context.Context, req *thothnetworkv1.DeleteStreamRequest) (*thothnetworkv1.DeleteStreamResponse, error) {
	s.adapter.logger.Debug("DeleteStream request received", "name", req.Name)

	// TODO: Implement stream deletion logic

	return nil, status.Error(codes.Unimplemented, "method DeleteStream not implemented")
}
