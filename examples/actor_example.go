package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ZanzyTHEbar/thothnetwork/internal/adapters/repositories/memory"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/device"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/room"
	actorservice "github.com/ZanzyTHEbar/thothnetwork/internal/services/actor"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/actor"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/google/uuid"
)

func main() {
	// Create logger
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		panic(err)
	}
	log.Info("Starting actor example")

	// Create repositories
	deviceRepo := memory.NewDeviceRepository()
	roomRepo := memory.NewRoomRepository()

	// Create actor system
	actorSystem := actor.NewActorSystem(actor.Config{
		Address:     "",  // No remote address for this example
		Port:        0,   // No port needed
		ClusterName: "",  // No cluster name needed
	}, log)

	// Create actor service
	actorService := actorservice.NewService(
		actorSystem,
		deviceRepo,
		roomRepo,
		log,
	)

	// Create context
	ctx := context.Background()

	// Create some test devices
	device1 := device.NewDevice(uuid.New().String(), "Device 1", "sensor")
	device1.Status = device.StatusOnline
	device1.Metadata["location"] = "living_room"

	device2 := device.NewDevice(uuid.New().String(), "Device 2", "actuator")
	device2.Status = device.StatusOnline
	device2.Metadata["location"] = "kitchen"

	// Register devices
	err = deviceRepo.Register(ctx, device1)
	if err != nil {
		log.Error("Failed to register device", "error", err)
		os.Exit(1)
	}

	err = deviceRepo.Register(ctx, device2)
	if err != nil {
		log.Error("Failed to register device", "error", err)
		os.Exit(1)
	}

	// Create a room
	room1 := &room.Room{
		ID:       uuid.New().String(),
		Name:     "Home",
		Type:     room.TypeMany2Many,
		Devices:  []string{},
		Metadata: map[string]string{"floor": "1"},
	}

	// Create room
	err = roomRepo.Create(ctx, room1)
	if err != nil {
		log.Error("Failed to create room", "error", err)
		os.Exit(1)
	}

	// Start device actors
	log.Info("Starting device actors")
	err = actorService.StartDeviceActor(ctx, device1.ID)
	if err != nil {
		log.Error("Failed to start device actor", "error", err)
		os.Exit(1)
	}

	err = actorService.StartDeviceActor(ctx, device2.ID)
	if err != nil {
		log.Error("Failed to start device actor", "error", err)
		os.Exit(1)
	}

	// Start room actor
	log.Info("Starting room actor")
	err = actorService.StartRoomActor(ctx, room1.ID)
	if err != nil {
		log.Error("Failed to start room actor", "error", err)
		os.Exit(1)
	}

	// Add devices to room
	log.Info("Adding devices to room")
	err = actorService.AddDeviceToRoom(ctx, room1.ID, device1.ID)
	if err != nil {
		log.Error("Failed to add device to room", "error", err)
		os.Exit(1)
	}

	err = actorService.AddDeviceToRoom(ctx, room1.ID, device2.ID)
	if err != nil {
		log.Error("Failed to add device to room", "error", err)
		os.Exit(1)
	}

	// Get room devices
	log.Info("Getting room devices")
	deviceIDs, err := actorService.GetRoomDevices(ctx, room1.ID)
	if err != nil {
		log.Error("Failed to get room devices", "error", err)
		os.Exit(1)
	}
	log.Info("Room devices", "devices", deviceIDs)

	// Send message to device
	log.Info("Sending message to device")
	msg := &message.Message{
		ID:      uuid.New().String(),
		Source:  "example",
		Target:  device1.ID,
		Type:    message.TypeCommand,
		Payload: []byte(`{"command": "turn_on"}`),
	}
	err = actorService.SendMessageToDevice(ctx, device1.ID, msg)
	if err != nil {
		log.Error("Failed to send message to device", "error", err)
		os.Exit(1)
	}

	// Send message to room
	log.Info("Sending message to room")
	roomMsg := &message.Message{
		ID:      uuid.New().String(),
		Source:  "example",
		Target:  room1.ID,
		Type:    message.TypeCommand,
		Payload: []byte(`{"command": "set_temperature", "value": 22}`),
	}
	err = actorService.SendMessageToRoom(ctx, room1.ID, roomMsg)
	if err != nil {
		log.Error("Failed to send message to room", "error", err)
		os.Exit(1)
	}

	// Wait for signals
	log.Info("System running. Press Ctrl+C to exit")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Keep the application running
	<-sigCh
	log.Info("Shutting down")

	// Stop all actors
	actorService.StopAllActors()

	// Stop actor system
	actorSystem.Stop()

	log.Info("Shutdown complete")
}
