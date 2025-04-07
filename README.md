# thothnetwork

thothnetwork is a highly scalable, real-time IoT backend system built in Golang, designed to manage and process data from millions of IoT devices. It leverages modern networking technologies such as NATs.io and/or libp2p to provide efficient communication, and it is optimized for concurrency, extensibility, and performance on Linux systems.

## Features

- **Device Management**: Register, update, and monitor IoT devices with digital twins and CRUD support
- **Real-time Streaming**: Process and publish data streams with sub-second latency
- **Scalable Networking**: Use NATs.io and libp2p for clustering, load balancing, and decentralized communication
- **Observability**: Includes logging, monitoring, and optional tracing for system insights
- **Extensibility**: Functions as a library, standalone binary, or containerized service with a CLI and socket interface
- **Advanced Analytics**: Support for graph-based analytics and integration with big data pipelines and LLMs

## Architecture

thothnetwork employs a **Hexagonal Architecture** (Ports and Adapters pattern) to ensure modularity and separation of concerns:

- **Core Domain**: Contains the business logic for device management, data processing, and communication
- **Ports**: Define standardized interfaces for interacting with external systems
- **Adapters**: Implement the ports for specific technologies or protocols

## Getting Started

### Prerequisites

- Go 1.21 or later
- NATS server (for messaging)
- Linux operating system

### Installation

```bash
# Clone the repository
git clone https://github.com/thothnetwork/thothnetwork.git
cd thothnetwork

# Build the server
go build -o thothnetwork-server ./cmd/server

# Build the CLI
go build -o thothnetwork-cli ./cmd/cli
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
./thothnetwork-server --config config.yaml
```

### Using the CLI

```bash
# List devices
./thothnetwork-cli device list

# Create a device
./thothnetwork-cli device create --name "Temperature Sensor" --type "sensor" --metadata '{"location":"room-1"}'

# Get device details
./thothnetwork-cli device get device-123

# Update a device
./thothnetwork-cli device update device-123 --name "Updated Sensor"

# Delete a device
./thothnetwork-cli device delete device-123
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
