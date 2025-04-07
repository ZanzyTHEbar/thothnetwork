package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ZanzyTHEbar/thothnetwork/internal/config"
	"github.com/ZanzyTHEbar/thothnetwork/internal/server"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create server instance
	s, err := server.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Create context
	ctx := context.Background()

	// Start server
	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	// Wait for termination signal
	sig := s.WaitForSignal()
	fmt.Printf("Received signal %v, shutting down...\n", sig)

	// Shutdown server
	if err := s.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server shutdown complete")
}
