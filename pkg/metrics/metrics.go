package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Metrics provides metrics collection functionality
type Metrics struct {
	registry *prometheus.Registry
	server   *http.Server
	logger   logger.Logger
	mu       sync.RWMutex

	// Metrics
	deviceCount        prometheus.Gauge
	messageCount       *prometheus.CounterVec
	messageLatency     *prometheus.HistogramVec
	connectionCount    *prometheus.GaugeVec
	pipelineLatency    *prometheus.HistogramVec
	pipelineErrorCount *prometheus.CounterVec
}

// Config holds configuration for metrics
type Config struct {
	Host string
	Port int
}

// NewMetrics creates a new metrics collector
func NewMetrics(config Config, logger logger.Logger) *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry: registry,
		logger:   logger.With("component", "metrics"),

		// Metrics
		deviceCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "thothnetwork_device_count",
			Help: "Number of registered devices",
		}),

		messageCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "thothnetwork_message_count",
				Help: "Number of messages processed",
			},
			[]string{"type", "source", "target"},
		),

		messageLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "thothnetwork_message_latency_seconds",
				Help:    "Message processing latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"type"},
		),

		connectionCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "thothnetwork_connection_count",
				Help: "Number of active connections",
			},
			[]string{"protocol"},
		),

		pipelineLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "thothnetwork_pipeline_latency_seconds",
				Help:    "Pipeline processing latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"pipeline"},
		),

		pipelineErrorCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "thothnetwork_pipeline_error_count",
				Help: "Number of pipeline processing errors",
			},
			[]string{"pipeline", "stage"},
		),
	}

	// Register metrics
	registry.MustRegister(m.deviceCount)
	registry.MustRegister(m.messageCount)
	registry.MustRegister(m.messageLatency)
	registry.MustRegister(m.connectionCount)
	registry.MustRegister(m.pipelineLatency)
	registry.MustRegister(m.pipelineErrorCount)

	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	m.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler: mux,
	}

	return m
}

// Start starts the metrics server
func (m *Metrics) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Start server in a goroutine
	go func() {
		m.logger.Info("Starting metrics server", "addr", m.server.Addr)

		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Metrics server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the metrics server
func (m *Metrics) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Shutdown server
	if m.server != nil {
		if err := m.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}

	return nil
}

// SetDeviceCount sets the device count
func (m *Metrics) SetDeviceCount(count int) {
	m.deviceCount.Set(float64(count))
}

// IncrementMessageCount increments the message count
func (m *Metrics) IncrementMessageCount(messageType, source, target string) {
	m.messageCount.WithLabelValues(messageType, source, target).Inc()
}

// ObserveMessageLatency observes message latency
func (m *Metrics) ObserveMessageLatency(messageType string, latency time.Duration) {
	m.messageLatency.WithLabelValues(messageType).Observe(latency.Seconds())
}

// SetConnectionCount sets the connection count
func (m *Metrics) SetConnectionCount(protocol string, count int) {
	m.connectionCount.WithLabelValues(protocol).Set(float64(count))
}

// ObservePipelineLatency observes pipeline latency
func (m *Metrics) ObservePipelineLatency(pipeline string, latency time.Duration) {
	m.pipelineLatency.WithLabelValues(pipeline).Observe(latency.Seconds())
}

// IncrementPipelineErrorCount increments the pipeline error count
func (m *Metrics) IncrementPipelineErrorCount(pipeline, stage string) {
	m.pipelineErrorCount.WithLabelValues(pipeline, stage).Inc()
}
