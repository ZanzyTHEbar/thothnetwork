package config

import (
	"time"

	errbuilder "github.com/ZanzyTHEbar/errbuilder-go"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	NATS      NATSConfig      `mapstructure:"nats"`
	LibP2P    LibP2PConfig    `mapstructure:"libp2p"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	HTTP      HTTPConfig      `mapstructure:"http"`
	MQTT      MQTTConfig      `mapstructure:"mqtt"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	GRPC      GRPCConfig      `mapstructure:"grpc"`
	Tracing   TracingConfig   `mapstructure:"tracing"`
	Actor     ActorConfig     `mapstructure:"actor"`
}

// ServerConfig holds configuration for the server
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// NATSConfig holds configuration for NATS
type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	Username      string        `mapstructure:"username"`
	Password      string        `mapstructure:"password"`
	Token         string        `mapstructure:"token"`
	MaxReconnects int           `mapstructure:"max_reconnects"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
	Timeout       time.Duration `mapstructure:"timeout"`
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

// HTTPConfig holds configuration for the HTTP adapter
type HTTPConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	BasePath        string        `mapstructure:"base_path"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	MaxHeaderBytes  int           `mapstructure:"max_header_bytes"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// MQTTConfig holds configuration for the MQTT adapter
type MQTTConfig struct {
	BrokerURL            string        `mapstructure:"broker_url"`
	ClientID             string        `mapstructure:"client_id"`
	Username             string        `mapstructure:"username"`
	Password             string        `mapstructure:"password"`
	CleanSession         bool          `mapstructure:"clean_session"`
	QoS                  byte          `mapstructure:"qos"`
	ConnectTimeout       time.Duration `mapstructure:"connect_timeout"`
	KeepAlive            time.Duration `mapstructure:"keep_alive"`
	PingTimeout          time.Duration `mapstructure:"ping_timeout"`
	ConnectRetryDelay    time.Duration `mapstructure:"connect_retry_delay"`
	MaxReconnectAttempts int           `mapstructure:"max_reconnect_attempts"`
	TopicPrefix          string        `mapstructure:"topic_prefix"`
}

// WebSocketConfig holds configuration for the WebSocket adapter
type WebSocketConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Path            string        `mapstructure:"path"`
	ReadBufferSize  int           `mapstructure:"read_buffer_size"`
	WriteBufferSize int           `mapstructure:"write_buffer_size"`
	PingInterval    time.Duration `mapstructure:"ping_interval"`
	PongWait        time.Duration `mapstructure:"pong_wait"`
}

// TracingConfig holds configuration for distributed tracing
type TracingConfig struct {
	ServiceName    string `mapstructure:"service_name"`
	ServiceVersion string `mapstructure:"service_version"`
	Endpoint       string `mapstructure:"endpoint"`
	Enabled        bool   `mapstructure:"enabled"`
}

// GRPCConfig holds configuration for gRPC
type GRPCConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// ActorConfig holds configuration for the actor system
type ActorConfig struct {
	Address     string `mapstructure:"address"`
	Port        int    `mapstructure:"port"`
	ClusterName string `mapstructure:"cluster_name"`
}

// Load loads the configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("/etc/thothnetwork")
	}
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, errbuilder.GenericErr("Failed to read config file", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, errbuilder.GenericErr("Failed to unmarshal config", err)
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
	viper.SetDefault("nats.max_reconnects", 10)
	viper.SetDefault("nats.reconnect_wait", 1*time.Second)
	viper.SetDefault("nats.timeout", 2*time.Second)

	// LibP2P defaults
	viper.SetDefault("libp2p.listen_addresses", []string{"/ip4/0.0.0.0/tcp/9000"})

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.host", "0.0.0.0")
	viper.SetDefault("metrics.port", 9090)

	// HTTP defaults
	viper.SetDefault("http.host", "0.0.0.0")
	viper.SetDefault("http.port", 8081)
	viper.SetDefault("http.base_path", "/api/v1")
	viper.SetDefault("http.read_timeout", 10*time.Second)
	viper.SetDefault("http.write_timeout", 10*time.Second)
	viper.SetDefault("http.max_header_bytes", 1<<20) // 1 MB
	viper.SetDefault("http.shutdown_timeout", 5*time.Second)

	// MQTT defaults
	viper.SetDefault("mqtt.broker_url", "tcp://localhost:1883")
	viper.SetDefault("mqtt.client_id", "thothnetwork-mqtt")
	viper.SetDefault("mqtt.qos", 1)
	viper.SetDefault("mqtt.connect_timeout", 10*time.Second)
	viper.SetDefault("mqtt.keep_alive", 30*time.Second)
	viper.SetDefault("mqtt.ping_timeout", 5*time.Second)
	viper.SetDefault("mqtt.connect_retry_delay", 5*time.Second)
	viper.SetDefault("mqtt.max_reconnect_attempts", 10)
	viper.SetDefault("mqtt.topic_prefix", "thothnetwork")

	// WebSocket defaults
	viper.SetDefault("websocket.host", "0.0.0.0")
	viper.SetDefault("websocket.port", 8082)
	viper.SetDefault("websocket.path", "/ws")
	viper.SetDefault("websocket.read_buffer_size", 1024)
	viper.SetDefault("websocket.write_buffer_size", 1024)
	viper.SetDefault("websocket.ping_interval", 30*time.Second)
	viper.SetDefault("websocket.pong_wait", 60*time.Second)

	// Tracing defaults
	viper.SetDefault("tracing.service_name", "thothnetwork")
	viper.SetDefault("tracing.service_version", "0.1.0")
	viper.SetDefault("tracing.endpoint", "http://localhost:14268/api/traces")
	viper.SetDefault("tracing.enabled", false)

	// gRPC defaults
	viper.SetDefault("grpc.host", "0.0.0.0")
	viper.SetDefault("grpc.port", 8083)

	// Actor system defaults
	viper.SetDefault("actor.address", "0.0.0.0")
	viper.SetDefault("actor.port", 8084)
	viper.SetDefault("actor.cluster_name", "thothnetwork-cluster")
}
