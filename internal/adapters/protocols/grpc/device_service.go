package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	thothnetworkv1 "github.com/ZanzyTHEbar/thothnetwork/pkg/gen/proto/thothnetwork/v1"
)

// DeviceService implements the DeviceService gRPC service
type DeviceService struct {
	thothnetworkv1.UnimplementedDeviceServiceServer
	adapter *Adapter
}

// NewDeviceService creates a new DeviceService instance
func NewDeviceService(adapter *Adapter) *DeviceService {
	return &DeviceService{
		adapter: adapter,
	}
}

// RegisterDevice registers a new device
func (s *DeviceService) RegisterDevice(ctx context.Context, req *thothnetworkv1.RegisterDeviceRequest) (*thothnetworkv1.RegisterDeviceResponse, error) {
	s.adapter.logger.Debug("RegisterDevice request received", "name", req.Name, "type", req.Type)

	// TODO: Implement actual device registration by interfacing with device service
	// This would typically involve:
	// 1. Validating the request
	// 2. Calling a domain service to register the device
	// 3. Converting the result to the response type

	// For now, return a mock response
	now := timestamppb.Now()
	device := &thothnetworkv1.Device{
		Id:        "mock-id", // In a real implementation, this would be generated or returned by a service
		Name:      req.Name,
		Type:      req.Type,
		Status:    thothnetworkv1.DeviceStatus_DEVICE_STATUS_ONLINE,
		Metadata:  req.Metadata,
		LastSeen:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return &thothnetworkv1.RegisterDeviceResponse{
		Id:     device.Id,
		Device: device,
	}, nil
}

// GetDevice retrieves a device by ID
func (s *DeviceService) GetDevice(ctx context.Context, req *thothnetworkv1.GetDeviceRequest) (*thothnetworkv1.GetDeviceResponse, error) {
	s.adapter.logger.Debug("GetDevice request received", "id", req.Id)

	// TODO: Implement actual device retrieval by interfacing with device service

	return nil, status.Error(codes.Unimplemented, "method GetDevice not implemented")
}

// UpdateDevice updates an existing device
func (s *DeviceService) UpdateDevice(ctx context.Context, req *thothnetworkv1.UpdateDeviceRequest) (*thothnetworkv1.UpdateDeviceResponse, error) {
	s.adapter.logger.Debug("UpdateDevice request received", "id", req.Id)

	// TODO: Implement actual device update by interfacing with device service

	return nil, status.Error(codes.Unimplemented, "method UpdateDevice not implemented")
}

// DeleteDevice deletes a device
func (s *DeviceService) DeleteDevice(ctx context.Context, req *thothnetworkv1.DeleteDeviceRequest) (*thothnetworkv1.DeleteDeviceResponse, error) {
	s.adapter.logger.Debug("DeleteDevice request received", "id", req.Id)

	// TODO: Implement actual device deletion by interfacing with device service

	return nil, status.Error(codes.Unimplemented, "method DeleteDevice not implemented")
}

// ListDevices lists devices with optional filtering
func (s *DeviceService) ListDevices(ctx context.Context, req *thothnetworkv1.ListDevicesRequest) (*thothnetworkv1.ListDevicesResponse, error) {
	s.adapter.logger.Debug("ListDevices request received")

	// TODO: Implement actual device listing by interfacing with device service

	return nil, status.Error(codes.Unimplemented, "method ListDevices not implemented")
}
