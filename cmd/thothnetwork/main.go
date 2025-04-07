package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZanzyTHEbar/thothnetwork/internal/adapters/brokers/nats"
	httpAdapter "github.com/ZanzyTHEbar/thothnetwork/internal/adapters/protocols/http"
	mqttAdapter "github.com/ZanzyTHEbar/thothnetwork/internal/adapters/protocols/mqtt"
	wsAdapter "github.com/ZanzyTHEbar/thothnetwork/internal/adapters/protocols/websocket"
	"github.com/ZanzyTHEbar/thothnetwork/internal/adapters/repositories/memory"
	"github.com/ZanzyTHEbar/thothnetwork/internal/config"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/message"
	"github.com/ZanzyTHEbar/thothnetwork/internal/core/pipeline"
	adapterService "github.com/ZanzyTHEbar/thothnetwork/internal/services/adapter"
	deviceService "github.com/ZanzyTHEbar/thothnetwork/internal/services/device"
	pipelineService "github.com/ZanzyTHEbar/thothnetwork/internal/services/pipeline"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/metrics"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/tracing"
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

	// Create logger
	log, err := logger.NewLogger(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: "stdout",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Create metrics collector
	metricsCollector := metrics.NewMetrics(metrics.Config{
		Host: cfg.Metrics.Host,
		Port: cfg.Metrics.Port,
	}, log)

	// Create tracer
	tracer, err := tracing.NewTracer(tracing.Config{
		ServiceName:    cfg.Tracing.ServiceName,
		ServiceVersion: cfg.Tracing.ServiceVersion,
		Endpoint:       cfg.Tracing.Endpoint,
		Enabled:        cfg.Tracing.Enabled,
	}, log)
	if err != nil {
		log.Error("Failed to create tracer", "error", err)
		os.Exit(1)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics server
	if err := metricsCollector.Start(ctx); err != nil {
		log.Error("Failed to start metrics server", "error", err)
		os.Exit(1)
	}

	// Create repositories
	deviceRepo := memory.NewDeviceRepository()
	twinRepo := memory.NewTwinRepository()
	// Room repository will be used in future implementations
	_ = memory.NewRoomRepository()

	// Create message broker
	broker := nats.NewMessageBroker(nats.Config{
		URL:           cfg.NATS.URL,
		Username:      cfg.NATS.Username,
		Password:      cfg.NATS.Password,
		Token:         cfg.NATS.Token,
		MaxReconnects: cfg.NATS.MaxReconnects,
		ReconnectWait: cfg.NATS.ReconnectWait,
		Timeout:       cfg.NATS.Timeout,
	}, log)
	if err := broker.Connect(ctx); err != nil {
		log.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer broker.Disconnect(ctx)

	// Create services
	// Device service will be used in future implementations
	_ = deviceService.NewService(deviceRepo, twinRepo, broker, log)
	pipelineSvc := pipelineService.NewService(broker, log)
	adapterSvc := adapterService.NewService(broker, log)

	// Create protocol adapters
	httpAdpt := httpAdapter.NewAdapter(httpAdapter.Config{
		Host:            cfg.HTTP.Host,
		Port:            cfg.HTTP.Port,
		BasePath:        cfg.HTTP.BasePath,
		ReadTimeout:     cfg.HTTP.ReadTimeout,
		WriteTimeout:    cfg.HTTP.WriteTimeout,
		MaxHeaderBytes:  cfg.HTTP.MaxHeaderBytes,
		ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}, log)

	mqttAdpt := mqttAdapter.NewAdapter(mqttAdapter.Config{
		BrokerURL:            cfg.MQTT.BrokerURL,
		ClientID:             cfg.MQTT.ClientID,
		Username:             cfg.MQTT.Username,
		Password:             cfg.MQTT.Password,
		CleanSession:         cfg.MQTT.CleanSession,
		QoS:                  cfg.MQTT.QoS,
		ConnectTimeout:       cfg.MQTT.ConnectTimeout,
		KeepAlive:            cfg.MQTT.KeepAlive,
		PingTimeout:          cfg.MQTT.PingTimeout,
		ConnectRetryDelay:    cfg.MQTT.ConnectRetryDelay,
		MaxReconnectAttempts: cfg.MQTT.MaxReconnectAttempts,
		TopicPrefix:          cfg.MQTT.TopicPrefix,
	}, log)

	wsAdpt := wsAdapter.NewAdapter(wsAdapter.Config{
		Host:            cfg.WebSocket.Host,
		Port:            cfg.WebSocket.Port,
		Path:            cfg.WebSocket.Path,
		ReadBufferSize:  cfg.WebSocket.ReadBufferSize,
		WriteBufferSize: cfg.WebSocket.WriteBufferSize,
		PingInterval:    cfg.WebSocket.PingInterval,
		PongWait:        cfg.WebSocket.PongWait,
	}, log)

	// Register adapters
	if err := adapterSvc.RegisterAdapter("http", httpAdpt); err != nil {
		log.Error("Failed to register HTTP adapter", "error", err)
		os.Exit(1)
	}
	if err := adapterSvc.RegisterAdapter("mqtt", mqttAdpt); err != nil {
		log.Error("Failed to register MQTT adapter", "error", err)
		os.Exit(1)
	}
	if err := adapterSvc.RegisterAdapter("websocket", wsAdpt); err != nil {
		log.Error("Failed to register WebSocket adapter", "error", err)
		os.Exit(1)
	}

	// Start adapters
	if err := adapterSvc.StartAdapter(ctx, "http"); err != nil {
		log.Error("Failed to start HTTP adapter", "error", err)
		os.Exit(1)
	}
	if err := adapterSvc.StartAdapter(ctx, "mqtt"); err != nil {
		log.Error("Failed to start MQTT adapter", "error", err)
		os.Exit(1)
	}
	if err := adapterSvc.StartAdapter(ctx, "websocket"); err != nil {
		log.Error("Failed to start WebSocket adapter", "error", err)
		os.Exit(1)
	}

	// Create default pipeline
	stages := []pipeline.ProcessingStage{
		pipelineSvc.CreateValidationStage("message-validation", func(ctx context.Context, msg *message.Message) error {
			// Validate message
			if msg.Source == "" {
				return errors.New("message source is required")
			}
			if msg.Target == "" {
				return errors.New("message target is required")
			}
			return nil
		}),
		pipelineSvc.CreateTransformationStage("message-transformation", func(ctx context.Context, msg *message.Message) (*message.Message, error) {
			// Transform message
			msg.SetMetadata("processed_at", time.Now().Format(time.RFC3339))
			return msg, nil
		}),
	}

	pipelineID, err := pipelineSvc.CreatePipeline(ctx, "default", "telemetry.>", "processed.telemetry", stages)
	if err != nil {
		log.Error("Failed to create default pipeline", "error", err)
		os.Exit(1)
	}

	// Start default pipeline
	if err := pipelineSvc.StartPipeline(ctx, pipelineID); err != nil {
		log.Error("Failed to start default pipeline", "error", err)
		os.Exit(1)
	}

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigCh
	log.Info("Received signal", "signal", sig)

	// Create shutdown context
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop adapters
	if err := adapterSvc.StopAdapter(shutdownCtx, "websocket"); err != nil {
		log.Error("Failed to stop WebSocket adapter", "error", err)
	}
	if err := adapterSvc.StopAdapter(shutdownCtx, "mqtt"); err != nil {
		log.Error("Failed to stop MQTT adapter", "error", err)
	}
	if err := adapterSvc.StopAdapter(shutdownCtx, "http"); err != nil {
		log.Error("Failed to stop HTTP adapter", "error", err)
	}

	// Stop pipeline
	if err := pipelineSvc.StopPipeline(shutdownCtx, pipelineID); err != nil {
		log.Error("Failed to stop default pipeline", "error", err)
	}

	// Stop metrics server
	if err := metricsCollector.Stop(shutdownCtx); err != nil {
		log.Error("Failed to stop metrics server", "error", err)
	}

	// Shutdown tracer
	if err := tracer.Shutdown(shutdownCtx); err != nil {
		log.Error("Failed to shutdown tracer", "error", err)
	}

	log.Info("Shutdown complete")
}
