.PHONY: build clean test run docker-build docker-run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME_SERVER=thothnetwork-server
BINARY_NAME_CLI=thothnetwork-cli
DOCKER_IMAGE=thothnetwork

all: test build

build: 
	$(GOBUILD) -o $(BINARY_NAME_SERVER) -v ./cmd/server
	$(GOBUILD) -o $(BINARY_NAME_CLI) -v ./cmd/cli

clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME_SERVER)
	rm -f $(BINARY_NAME_CLI)

test: 
	$(GOTEST) -v ./...

run: build
	./$(BINARY_NAME_SERVER)

deps:
	$(GOMOD) download

tidy:
	$(GOMOD) tidy

docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	docker run --rm -p 8080:8080 -p 9090:9090 $(DOCKER_IMAGE)

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

lint:
	golangci-lint run ./...

proto:
	buf generate

# Generate mocks for testing
mocks:
	mockery --all --dir=internal --output=internal/mocks

# Run server with hot reload
dev:
	air -c .air.toml
