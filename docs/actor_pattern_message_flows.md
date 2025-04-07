# 4. Message Flow Diagrams

## Overview

This section illustrates the key message flows in the actor-based IoT system. These diagrams show how different components interact through message passing to accomplish various tasks.

## Device Registration and Actor Creation

This diagram shows the process of registering a new device and creating a corresponding device actor.

```mermaid
sequenceDiagram
    participant Client
    participant DeviceService
    participant DeviceRepo
    participant ActorService
    participant ActorSystem
    participant DeviceActor
    
    Client->>DeviceService: RegisterDevice(device)
    DeviceService->>DeviceRepo: Register(device)
    DeviceRepo-->>DeviceService: Success
    DeviceService->>ActorService: StartDeviceActor(deviceID)
    ActorService->>DeviceRepo: Get(deviceID)
    DeviceRepo-->>ActorService: device
    ActorService->>ActorSystem: SpawnDevice(deviceID)
    ActorSystem->>DeviceActor: Create
    DeviceActor-->>ActorSystem: PID
    ActorSystem-->>ActorService: PID
    ActorService->>ActorService: Store PID
    ActorService->>DeviceActor: UpdateStateCommand(device)
    DeviceActor-->>ActorService: UpdateStateResponse
    ActorService-->>DeviceService: Success
    DeviceService-->>Client: deviceID
```

### Process Description:

1. The client sends a request to register a new device to the DeviceService.
2. The DeviceService registers the device in the DeviceRepo.
3. The DeviceService requests the ActorService to start a device actor for the new device.
4. The ActorService retrieves the device details from the DeviceRepo.
5. The ActorService requests the ActorSystem to spawn a new device actor.
6. The ActorSystem creates a new DeviceActor and returns its PID (Process ID).
7. The ActorService stores the PID for future reference.
8. The ActorService sends an UpdateStateCommand to the DeviceActor to initialize its state.
9. The DeviceActor updates its state and responds with success.
10. The ActorService reports success to the DeviceService, which in turn responds to the client.

## Room Creation and Device Addition

This diagram shows the process of creating a new room and adding a device to it.

```mermaid
sequenceDiagram
    participant Client
    participant RoomService
    participant RoomRepo
    participant ActorService
    participant ActorSystem
    participant RoomActor
    participant DeviceActor
    
    Client->>RoomService: CreateRoom(room)
    RoomService->>RoomRepo: Create(room)
    RoomRepo-->>RoomService: Success
    RoomService->>ActorService: StartRoomActor(roomID)
    ActorService->>RoomRepo: Get(roomID)
    RoomRepo-->>ActorService: room
    ActorService->>ActorSystem: SpawnRoom(roomID)
    ActorSystem->>RoomActor: Create
    RoomActor-->>ActorSystem: PID
    ActorSystem-->>ActorService: PID
    ActorService->>ActorService: Store PID
    
    Client->>RoomService: AddDeviceToRoom(roomID, deviceID)
    RoomService->>ActorService: AddDeviceToRoom(roomID, deviceID)
    ActorService->>RoomRepo: AddDevice(roomID, deviceID)
    RoomRepo-->>ActorService: Success
    ActorService->>ActorService: Get DevicePID
    ActorService->>RoomActor: AddDeviceCommand(deviceID, devicePID)
    RoomActor-->>ActorService: AddDeviceResponse
    ActorService-->>RoomService: Success
    RoomService-->>Client: Success
```

### Process Description:

1. The client sends a request to create a new room to the RoomService.
2. The RoomService creates the room in the RoomRepo.
3. The RoomService requests the ActorService to start a room actor for the new room.
4. The ActorService retrieves the room details from the RoomRepo.
5. The ActorService requests the ActorSystem to spawn a new room actor.
6. The ActorSystem creates a new RoomActor and returns its PID.
7. The ActorService stores the PID for future reference.
8. The client sends a request to add a device to the room.
9. The RoomService forwards the request to the ActorService.
10. The ActorService adds the device to the room in the RoomRepo.
11. The ActorService retrieves the device actor's PID.
12. The ActorService sends an AddDeviceCommand to the RoomActor.
13. The RoomActor adds the device and responds with success.
14. The ActorService reports success to the RoomService, which in turn responds to the client.

## Message Sending to Device

This diagram shows the process of sending a message to a device.

```mermaid
sequenceDiagram
    participant Client
    participant MessageService
    participant ActorService
    participant DeviceActor
    
    Client->>MessageService: SendMessageToDevice(deviceID, message)
    MessageService->>ActorService: SendMessageToDevice(deviceID, message)
    ActorService->>ActorService: Get DevicePID
    ActorService->>DeviceActor: Message
    DeviceActor->>DeviceActor: Process Message
    DeviceActor-->>Client: Response (if needed)
```

### Process Description:

1. The client sends a message to a device through the MessageService.
2. The MessageService forwards the message to the ActorService.
3. The ActorService retrieves the device actor's PID.
4. The ActorService sends the message to the DeviceActor.
5. The DeviceActor processes the message.
6. If a response is needed, the DeviceActor sends it back to the client.

## Message Broadcasting to Room

This diagram shows the process of broadcasting a message to all devices in a room.

```mermaid
sequenceDiagram
    participant Client
    participant MessageService
    participant ActorService
    participant RoomActor
    participant DeviceActor1
    participant DeviceActor2
    
    Client->>MessageService: SendMessageToRoom(roomID, message)
    MessageService->>ActorService: SendMessageToRoom(roomID, message)
    ActorService->>ActorService: Get RoomPID
    ActorService->>RoomActor: Message
    RoomActor->>RoomActor: Get Device PIDs
    RoomActor->>DeviceActor1: Forward Message
    RoomActor->>DeviceActor2: Forward Message
    DeviceActor1->>DeviceActor1: Process Message
    DeviceActor2->>DeviceActor2: Process Message
```

### Process Description:

1. The client sends a message to a room through the MessageService.
2. The MessageService forwards the message to the ActorService.
3. The ActorService retrieves the room actor's PID.
4. The ActorService sends the message to the RoomActor.
5. The RoomActor retrieves the PIDs of all device actors in the room.
6. The RoomActor forwards the message to each device actor.
7. Each device actor processes the message.

## Telemetry Processing Flow

This diagram shows the process of processing telemetry data from a device.

```mermaid
sequenceDiagram
    participant Device
    participant MQTT
    participant MessageBroker
    participant ActorService
    participant DeviceActor
    participant TwinActor
    participant PipelineActor
    
    Device->>MQTT: Publish Telemetry
    MQTT->>MessageBroker: Forward Telemetry
    MessageBroker->>ActorService: Forward Telemetry
    ActorService->>DeviceActor: Send Telemetry
    DeviceActor->>DeviceActor: Update State
    DeviceActor->>TwinActor: Forward Telemetry
    TwinActor->>TwinActor: Update State
    DeviceActor->>PipelineActor: Forward Telemetry
    PipelineActor->>PipelineActor: Process Telemetry
```

### Process Description:

1. The physical device publishes telemetry data to the MQTT broker.
2. The MQTT broker forwards the telemetry data to the message broker.
3. The message broker forwards the telemetry data to the ActorService.
4. The ActorService sends the telemetry data to the appropriate DeviceActor.
5. The DeviceActor updates its state based on the telemetry data.
6. The DeviceActor forwards the telemetry data to the corresponding TwinActor.
7. The TwinActor updates its state based on the telemetry data.
8. The DeviceActor forwards the telemetry data to the PipelineActor for processing.
9. The PipelineActor processes the telemetry data through its stages.

## Command Execution Flow

This diagram shows the process of executing a command on a device.

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant MessageBroker
    participant ActorService
    participant DeviceActor
    participant Device
    
    Client->>API: Send Command
    API->>MessageBroker: Publish Command
    MessageBroker->>ActorService: Forward Command
    ActorService->>DeviceActor: Send Command
    DeviceActor->>DeviceActor: Process Command
    DeviceActor->>Device: Execute Command
    Device-->>DeviceActor: Command Result
    DeviceActor-->>ActorService: Command Response
    ActorService-->>MessageBroker: Publish Response
    MessageBroker-->>API: Forward Response
    API-->>Client: Send Response
```

### Process Description:

1. The client sends a command to the API.
2. The API publishes the command to the message broker.
3. The message broker forwards the command to the ActorService.
4. The ActorService sends the command to the appropriate DeviceActor.
5. The DeviceActor processes the command.
6. The DeviceActor executes the command on the physical device.
7. The physical device returns the command result to the DeviceActor.
8. The DeviceActor sends the command response to the ActorService.
9. The ActorService publishes the response to the message broker.
10. The message broker forwards the response to the API.
11. The API sends the response to the client.

## Actor System Startup and Shutdown

This diagram shows the process of starting up and shutting down the actor system.

```mermaid
sequenceDiagram
    participant App
    participant ActorService
    participant ActorSystem
    participant DeviceRepo
    participant RoomRepo
    participant DeviceActor
    participant RoomActor
    
    App->>ActorSystem: NewActorSystem(config)
    ActorSystem-->>App: actorSystem
    
    App->>ActorService: NewService(actorSystem, repos)
    ActorService-->>App: service
    
    App->>ActorService: StartAllDeviceActors()
    ActorService->>DeviceRepo: List()
    DeviceRepo-->>ActorService: devices
    loop For each device
        ActorService->>ActorSystem: SpawnDevice(deviceID)
        ActorSystem->>DeviceActor: Create
        DeviceActor-->>ActorSystem: PID
        ActorSystem-->>ActorService: PID
        ActorService->>ActorService: Store PID
    end
    
    App->>ActorService: StartAllRoomActors()
    ActorService->>RoomRepo: List()
    RoomRepo-->>ActorService: rooms
    loop For each room
        ActorService->>ActorSystem: SpawnRoom(roomID)
        ActorSystem->>RoomActor: Create
        RoomActor-->>ActorSystem: PID
        ActorSystem-->>ActorService: PID
        ActorService->>ActorService: Store PID
        
        ActorService->>RoomRepo: ListDevices(roomID)
        RoomRepo-->>ActorService: deviceIDs
        loop For each device in room
            ActorService->>RoomActor: AddDeviceCommand(deviceID, devicePID)
            RoomActor-->>ActorService: Success
        end
    end
    
    App->>ActorService: StopAllActors()
    loop For each device actor
        ActorService->>ActorSystem: Stop(devicePID)
        ActorSystem->>DeviceActor: Stop
        DeviceActor-->>ActorSystem: Stopped
    end
    
    loop For each room actor
        ActorService->>ActorSystem: Stop(roomPID)
        ActorSystem->>RoomActor: Stop
        RoomActor-->>ActorSystem: Stopped
    end
    
    App->>ActorSystem: Stop()
```

### Process Description:

1. The application creates a new actor system.
2. The application creates a new actor service with the actor system and repositories.
3. The application requests the actor service to start all device actors.
4. The actor service retrieves all devices from the device repository.
5. For each device, the actor service spawns a device actor and stores its PID.
6. The application requests the actor service to start all room actors.
7. The actor service retrieves all rooms from the room repository.
8. For each room, the actor service spawns a room actor and stores its PID.
9. For each room, the actor service retrieves the devices in the room and adds them to the room actor.
10. When shutting down, the application requests the actor service to stop all actors.
11. The actor service stops all device actors and room actors.
12. The application stops the actor system.
