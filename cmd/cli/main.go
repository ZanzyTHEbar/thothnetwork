package main

import (
	"os"

	"github.com/ZanzyTHEbar/thothnetwork/cmd/cli/commands"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "thothnetwork",
	Short: "thothnetwork is a highly scalable IoT backend system",
	Long: `thothnetwork is a highly scalable, real-time IoT backend system built in Golang,
designed to manage and process data from millions of IoT devices.`,
	Version: "0.1.0",
}

func init() {
	// Add commands
	rootCmd.AddCommand(commands.NewDeviceCommand())

	// Add global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().StringP("server", "s", "http://localhost:8080", "Server URL")
}

func main() {
	// Initialize logger
	log := logger.NewLogger()

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		log.Error("Failed to execute command", "error", err)
		os.Exit(1)
	}
}
