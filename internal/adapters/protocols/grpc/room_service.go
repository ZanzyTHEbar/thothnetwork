package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	thothnetworkv1 "github.com/ZanzyTHEbar/thothnetwork/pkg/gen/proto/thothnetwork/v1"
)

// RoomService implements the RoomService gRPC service
type RoomService struct {
	thothnetworkv1.UnimplementedRoomServiceServer
	adapter *Adapter
}

// NewRoomService creates a new RoomService instance
func NewRoomService(adapter *Adapter) *RoomService {
	return &RoomService{
		adapter: adapter,
	}
}

// CreateRoom creates a new room
func (s *RoomService) CreateRoom(ctx context.Context, req *thothnetworkv1.CreateRoomRequest) (*thothnetworkv1.CreateRoomResponse, error) {
	s.adapter.logger.Debug("CreateRoom request received", "name", req.Name, "type", req.Type)

	// TODO: Implement actual room creation logic
	// This would typically involve:
	// 1. Validating the request
	// 2. Calling a domain service to create the room
	// 3. Converting the result to the response type

	// For now, return a mock response
	now := timestamppb.Now()
	room := &thothnetworkv1.Room{
		Id:        "mock-room-id",
		Name:      req.Name,
		Type:      req.Type,
		Devices:   req.Devices,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return &thothnetworkv1.CreateRoomResponse{
		Id:   room.Id,
		Room: room,
	}, nil
}

// GetRoom retrieves a room by ID
func (s *RoomService) GetRoom(ctx context.Context, req *thothnetworkv1.GetRoomRequest) (*thothnetworkv1.GetRoomResponse, error) {
	s.adapter.logger.Debug("GetRoom request received", "id", req.Id)

	// TODO: Implement actual room retrieval logic

	return nil, status.Error(codes.Unimplemented, "method GetRoom not implemented")
}

// UpdateRoom updates an existing room
func (s *RoomService) UpdateRoom(ctx context.Context, req *thothnetworkv1.UpdateRoomRequest) (*thothnetworkv1.UpdateRoomResponse, error) {
	s.adapter.logger.Debug("UpdateRoom request received", "id", req.Id)

	// TODO: Implement actual room update logic

	return nil, status.Error(codes.Unimplemented, "method UpdateRoom not implemented")
}

// DeleteRoom deletes a room
func (s *RoomService) DeleteRoom(ctx context.Context, req *thothnetworkv1.DeleteRoomRequest) (*thothnetworkv1.DeleteRoomResponse, error) {
	s.adapter.logger.Debug("DeleteRoom request received", "id", req.Id)

	// TODO: Implement actual room deletion logic

	return nil, status.Error(codes.Unimplemented, "method DeleteRoom not implemented")
}

// ListRooms lists rooms with optional filtering
func (s *RoomService) ListRooms(ctx context.Context, req *thothnetworkv1.ListRoomsRequest) (*thothnetworkv1.ListRoomsResponse, error) {
	s.adapter.logger.Debug("ListRooms request received")

	// TODO: Implement actual room listing logic

	return nil, status.Error(codes.Unimplemented, "method ListRooms not implemented")
}

// AddDeviceToRoom adds a device to a room
func (s *RoomService) AddDeviceToRoom(ctx context.Context, req *thothnetworkv1.AddDeviceToRoomRequest) (*thothnetworkv1.AddDeviceToRoomResponse, error) {
	s.adapter.logger.Debug("AddDeviceToRoom request received", "roomID", req.RoomId, "deviceID", req.DeviceId)

	// TODO: Implement actual logic to add a device to a room

	return nil, status.Error(codes.Unimplemented, "method AddDeviceToRoom not implemented")
}

// RemoveDeviceFromRoom removes a device from a room
func (s *RoomService) RemoveDeviceFromRoom(ctx context.Context, req *thothnetworkv1.RemoveDeviceFromRoomRequest) (*thothnetworkv1.RemoveDeviceFromRoomResponse, error) {
	s.adapter.logger.Debug("RemoveDeviceFromRoom request received", "roomID", req.RoomId, "deviceID", req.DeviceId)

	// TODO: Implement actual logic to remove a device from a room

	return nil, status.Error(codes.Unimplemented, "method RemoveDeviceFromRoom not implemented")
}

// ListDevicesInRoom lists devices in a room
func (s *RoomService) ListDevicesInRoom(ctx context.Context, req *thothnetworkv1.ListDevicesInRoomRequest) (*thothnetworkv1.ListDevicesInRoomResponse, error) {
	s.adapter.logger.Debug("ListDevicesInRoom request received", "roomID", req.RoomId)

	// TODO: Implement actual logic to list devices in a room

	return nil, status.Error(codes.Unimplemented, "method ListDevicesInRoom not implemented")
}

// FindRoomsForDevice finds rooms for a device
func (s *RoomService) FindRoomsForDevice(ctx context.Context, req *thothnetworkv1.FindRoomsForDeviceRequest) (*thothnetworkv1.FindRoomsForDeviceResponse, error) {
	s.adapter.logger.Debug("FindRoomsForDevice request received", "deviceID", req.DeviceId)

	// TODO: Implement actual logic to find rooms for a device

	return nil, status.Error(codes.Unimplemented, "method FindRoomsForDevice not implemented")
}
