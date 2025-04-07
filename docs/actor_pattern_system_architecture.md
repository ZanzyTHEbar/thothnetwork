# 2. System Architecture

## Overview

The system follows a hexagonal architecture with the actor pattern integrated as a core component for handling concurrency and message passing. This architecture separates the core domain logic from external concerns, making the system more maintainable and testable.

## Architectural Diagram

```mermaid
flowchart TB
    subgraph "Core Domain"
        Device["Device Domain"]
        Room["Room Domain"]
        Message["Message Domain"]
        Twin["Digital Twin Domain"]
        Pipeline["Pipeline Domain"]
    end
    
    subgraph "Actor System"
        ActorSystem["Actor System"]
        DeviceActor["Device Actor"]
        TwinActor["Twin Actor"]
        RoomActor["Room Actor"]
        PipelineActor["Pipeline Actor"]
    end
    
    subgraph "Ports"
        Repositories["Repository Interfaces"]
        Brokers["Message Broker Interfaces"]
    end
    
    subgraph "Adapters"
        MemoryRepo["Memory Repositories"]
        NATSBroker["NATS Message Broker"]
    end
    
    subgraph "Services"
        ActorService["Actor Service"]
        DeviceService["Device Service"]
        RoomService["Room Service"]
    end
    
    subgraph "API Layer"
        REST["REST API"]
        MQTT["MQTT API"]
    end
    
    Device --> DeviceActor
    Room --> RoomActor
    Twin --> TwinActor
    Pipeline --> PipelineActor
    
    DeviceActor --> ActorSystem
    TwinActor --> ActorSystem
    RoomActor --> ActorSystem
    PipelineActor --> ActorSystem
    
    ActorSystem --> ActorService
    
    ActorService --> Repositories
    DeviceService --> Repositories
    RoomService --> Repositories
    
    Repositories --> MemoryRepo
    Brokers --> NATSBroker
    
    ActorService --> API Layer
    DeviceService --> API Layer
    RoomService --> API Layer
```

## Architectural Layers

### Core Domain

The core domain contains the business entities and logic of the IoT system:

- **Device Domain**: Represents physical IoT devices, their properties, and behaviors.
- **Room Domain**: Represents physical or logical rooms containing multiple devices.
- **Message Domain**: Defines the structure and types of messages exchanged in the system.
- **Twin Domain**: Represents digital twins of physical devices.
- **Pipeline Domain**: Defines data processing pipelines for transforming and analyzing device data.

### Actor System

The actor system provides the concurrency model for the IoT system:

- **Actor System**: The central coordinator that manages actor creation, messaging, and lifecycle.
- **Device Actor**: Represents a physical IoT device and handles device-specific messages and state.
- **Twin Actor**: Represents a digital twin of a physical device.
- **Room Actor**: Represents a room containing multiple devices.
- **Pipeline Actor**: Represents a processing pipeline for data transformation and analysis.

### Ports

The ports define the interfaces that the core domain uses to interact with external systems:

- **Repository Interfaces**: Define how the core domain interacts with data storage.
- **Message Broker Interfaces**: Define how the core domain exchanges messages with external systems.

### Adapters

The adapters implement the port interfaces to connect the core domain to external systems:

- **Memory Repositories**: In-memory implementations of the repository interfaces.
- **NATS Message Broker**: Implementation of the message broker interface using NATS.

### Services

The services provide high-level operations that coordinate multiple domain entities:

- **Actor Service**: Manages the lifecycle of actors and provides a high-level API for interacting with the actor system.
- **Device Service**: Provides operations for managing devices.
- **Room Service**: Provides operations for managing rooms and the devices within them.

### API Layer

The API layer exposes the system's functionality to external clients:

- **REST API**: HTTP-based API for web and mobile clients.
- **MQTT API**: MQTT-based API for IoT devices.

## Key Architectural Decisions

1. **Hexagonal Architecture**: We chose a hexagonal architecture to separate the core domain logic from external concerns, making the system more maintainable and testable.

2. **Actor Pattern**: We implemented the actor pattern using the `go-hollywood` package to handle concurrency and message passing in a scalable and fault-tolerant way.

3. **Domain-Driven Design**: We used domain-driven design principles to model the core domain entities and their relationships.

4. **Message-Driven Communication**: We used message-driven communication between components to reduce coupling and improve scalability.

5. **Repository Pattern**: We used the repository pattern to abstract data storage operations from the core domain logic.

## Benefits of the Architecture

1. **Separation of Concerns**: The hexagonal architecture separates the core domain logic from external concerns, making the system more maintainable.

2. **Testability**: The use of interfaces and dependency injection makes the system more testable.

3. **Flexibility**: The system can easily adapt to changes in external systems by implementing new adapters.

4. **Scalability**: The actor pattern provides a natural way to scale the system horizontally.

5. **Fault Tolerance**: The actor pattern includes supervision strategies for handling failures gracefully.

## Limitations and Considerations

1. **Complexity**: The actor pattern adds some complexity to the system, which may increase the learning curve for new developers.

2. **Message Serialization**: When actors communicate across node boundaries, messages must be serializable, which may limit the types of data that can be exchanged.

3. **Debugging**: Debugging actor-based systems can be challenging due to their asynchronous nature.

4. **State Management**: Actors maintain state, which must be carefully managed to avoid inconsistencies.

5. **Performance Overhead**: The actor pattern introduces some performance overhead due to message passing and actor creation/destruction.
