package config

import (
	"github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	NATS    NATSConfig    `mapstructure:"nats"`
	LibP2P  LibP2PConfig  `mapstructure:"libp2p"`
	Logging LoggingConfig `mapstructure:"logging"`
	Metrics MetricsConfig `mapstructure:"metrics"`
}

// ServerConfig holds configuration for the server
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// NATSConfig holds configuration for NATS
type NATSConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Token    string `mapstructure:"token"`
}

// LibP2PConfig holds configuration for libp2p
type LibP2PConfig struct {
	ListenAddresses []string `mapstructure:"listen_addresses"`
	BootstrapPeers  []string `mapstructure:"bootstrap_peers"`
}

// LoggingConfig holds configuration for logging
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// MetricsConfig holds configuration for metrics
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// LoadConfig loads the configuration from file and environment variables
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/thothnetwork")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, errbuilder.New().WithMessage("Failed to read config file").WithError(err).Build()
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, errbuilder.New().WithMessage("Failed to unmarshal config").WithError(err).Build()
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")

	// LibP2P defaults
	viper.SetDefault("libp2p.listen_addresses", []string{"/ip4/0.0.0.0/tcp/9000"})

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.host", "0.0.0.0")
	viper.SetDefault("metrics.port", 9090)
}
