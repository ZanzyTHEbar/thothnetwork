package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ZanzyTHEbar/thothnetwork/internal/config"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

func main() {
	// Initialize logger
	log := logger.NewDefaultLogger()
	log.Info("Starting thothnetwork server...")

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatal("Failed to load configuration", "error", err)
	}

	// Create context that listens for the interrupt signal from the OS
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create signal channel to listen for termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		// TODO: Initialize and start server components
		log.Info("Server started", "config", cfg)
	}()

	// Block until we receive a termination signal
	<-sigCh
	log.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	// TODO: Implement graceful shutdown of all components

	log.Info("Server gracefully stopped")
}
