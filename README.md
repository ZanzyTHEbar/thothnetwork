<!-- ![Thoth Network Logo](/docs/thothnetwork.png) -->
<img src="/docs/thothnetwork.png" alt="Thoth Network Logo" width="200" height="200"></img>

# Thoth Network

> [!IMPORTANT]\
> This project is currently in active development and is not yet ready for production use.

Thoth Network is a highly scalable, real-time IoT backend system built in Golang, designed to manage and process data from millions of IoT devices. It leverages modern networking technologies such as NATs.io and/or libp2p to provide efficient communication. It is optimized for concurrency, extensibility, and performance on Linux systems.

This system was inspired by ROS2, MQTT, and other IoT platforms, but it is designed to be more efficient, scalable, and future-proof.

I originally intended this system to be the backbone of a highly-decentralized cloud-battery-management-system. However, the project has grown to be a more general-purpose IoT backend system.

I had some fun and used it as a decentralized backbone for multi-agent tasks across the internet. Very neat potentials there.

## Features

- **Device Management**: Register, update, and monitor IoT devices with digital twins and CRUD support
- **Real-time Streaming**: Process and publish data streams with sub-second latency
- **Scalable Networking**: Use NATs.io and libp2p for clustering, load balancing, and decentralized communication
- **Observability**: Includes logging, monitoring, and optional tracing for system insights
- **Extensibility**: Functions as a library, standalone binary, or containerized service with a CLI and socket interface
- **Advanced Analytics**: Support for graph-based analytics and integration with big data pipelines and LLMs
- **Security**: Built-in support for authentication, authorization, and encryption using X.509 certificates & JWT tokens
  - Potential future implementation of Zero Trust Network

## Architecture

Thoth Network employs a **Hexagonal Architecture** (Ports and Adapters pattern) to ensure modularity and separation of concerns:

- **Core Domain**: Contains the business logic for device management, data processing, and communication
- **Ports**: Define standardized interfaces for interacting with external systems
- **Adapters**: Implement the ports for specific technologies or protocols

### Project Structure

- **cmd/**: Contains the main entry points for the application
  - **cmd/server/**: The server application that runs the Thoth Network backend
  - **cmd/client/**: Command-line interface for interacting with the Thoth Network server
- **internal/**: Internal packages not meant for external use
  - **internal/server/**: The main server implementation
  - **internal/config/**: Configuration handling
  - **internal/core/**: Core domain models and business logic
  - **internal/ports/**: Interface definitions for external systems
  - **internal/adapters/**: Implementations of the port interfaces
  - **internal/services/**: Business logic services
- **pkg/**: Reusable packages that can be used by external applications
- **api/**: API definitions (Protocol Buffers, OpenAPI, etc.)
- **docs/**: Documentation

## Getting Started

### Prerequisites

- Go 1.24 or later
- NATS server (for messaging)
- Linux operating system

> [!NOTE]\
> There is no plan to support windows or macosx into the foreseeable future.
> For a multitude of reasons.

### Installation

```bash
# Clone the repository
git clone https://github.com/ZanzyTHEbar/thothnetwork.git
cd thothnetwork

# Build the server
go build -o thothnetwork-server cmd/server

# Build the CLI
go build -o thothnetwork-client cmd/client
```

### Configuration

Create a `config.yaml` file:

```yaml
server:
  host: 0.0.0.0
  port: 8080

nats:
  url: nats://localhost:4222
  username: ""
  password: ""
  token: ""
  certificate: ""
  max_reconnects: 10
  reconnect_wait: 1s
  timeout: 2s

logging:
  level: info
  format: json

metrics:
  enabled: true
  host: 0.0.0.0
  port: 9090
```

### Running the Server

```bash
thothnetwork-server --config config.yaml
```

### Using the Client

```bash
# List devices
thothnetwork-client device list

# Create a device
thothnetwork-client device create --name "Temperature Sensor" --type "sensor" --metadata '{"location":"room-1"}'

# Get device details
thothnetwork-client device get device-123

# Update a device
thothnetwork-client device update device-123 --name "Updated Sensor"

# Delete a device
thothnetwork-client device delete device-123
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
