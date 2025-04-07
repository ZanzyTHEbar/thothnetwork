package actor

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// ActorMetrics collects metrics for the actor system
type ActorMetrics struct {
	// Registry for metrics
	registry *prometheus.Registry

	// Actor counts
	deviceActorCount   prometheus.Gauge
	twinActorCount     prometheus.Gauge
	roomActorCount     prometheus.Gauge
	pipelineActorCount prometheus.Gauge
	totalActorCount    prometheus.Gauge

	// Message metrics
	messageCount        *prometheus.CounterVec
	messageLatency      *prometheus.HistogramVec
	messageErrorCount   *prometheus.CounterVec
	messageSize         *prometheus.HistogramVec
	messageQueueSize    *prometheus.GaugeVec
	messageProcessTime  *prometheus.HistogramVec

	// Actor lifecycle metrics
	actorStartCount     *prometheus.CounterVec
	actorStopCount      *prometheus.CounterVec
	actorRestartCount   *prometheus.CounterVec
	actorPassivateCount *prometheus.CounterVec
	actorActivateCount  *prometheus.CounterVec
	actorErrorCount     *prometheus.CounterVec
	actorLifetime       *prometheus.HistogramVec

	// Supervision metrics
	supervisionRestartCount *prometheus.CounterVec
	supervisionStopCount    *prometheus.CounterVec
	supervisionErrorCount   *prometheus.CounterVec

	// Circuit breaker metrics
	circuitBreakerOpenCount   *prometheus.CounterVec
	circuitBreakerCloseCount  *prometheus.CounterVec
	circuitBreakerRejectCount *prometheus.CounterVec
	circuitBreakerState       *prometheus.GaugeVec

	// Resource metrics
	memoryUsage *prometheus.GaugeVec
	cpuUsage    *prometheus.GaugeVec

	// Logger
	logger logger.Logger

	// Mutex for thread safety
	mu sync.RWMutex
}

// NewActorMetrics creates a new actor metrics collector
func NewActorMetrics(logger logger.Logger) *ActorMetrics {
	registry := prometheus.NewRegistry()

	metrics := &ActorMetrics{
		registry: registry,
		logger:   logger.With("component", "actor_metrics"),
	}

	// Initialize metrics
	metrics.initializeMetrics()

	// Register metrics with the registry
	metrics.registerMetrics()

	return metrics
}

// initializeMetrics initializes all metrics
func (m *ActorMetrics) initializeMetrics() {
	// Actor counts
	m.deviceActorCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "actor_device_count",
		Help: "Number of device actors",
	})

	m.twinActorCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "actor_twin_count",
		Help: "Number of twin actors",
	})

	m.roomActorCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "actor_room_count",
		Help: "Number of room actors",
	})

	m.pipelineActorCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "actor_pipeline_count",
		Help: "Number of pipeline actors",
	})

	m.totalActorCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "actor_total_count",
		Help: "Total number of actors",
	})

	// Message metrics
	m.messageCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_message_count",
		Help: "Number of messages processed by actors",
	}, []string{"actor_type", "message_type"})

	m.messageLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "actor_message_latency_seconds",
		Help:    "Latency of message processing in seconds",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
	}, []string{"actor_type", "message_type"})

	m.messageErrorCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_message_error_count",
		Help: "Number of errors during message processing",
	}, []string{"actor_type", "message_type", "error_type"})

	m.messageSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "actor_message_size_bytes",
		Help:    "Size of messages in bytes",
		Buckets: prometheus.ExponentialBuckets(10, 10, 6), // 10B to 1MB
	}, []string{"actor_type", "message_type"})

	m.messageQueueSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "actor_message_queue_size",
		Help: "Size of actor message queues",
	}, []string{"actor_type", "actor_id"})

	m.messageProcessTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "actor_message_process_time_seconds",
		Help:    "Time to process a message in seconds",
		Buckets: prometheus.ExponentialBuckets(0.0001, 10, 6), // 0.1ms to 100s
	}, []string{"actor_type", "message_type"})

	// Actor lifecycle metrics
	m.actorStartCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_start_count",
		Help: "Number of actor starts",
	}, []string{"actor_type"})

	m.actorStopCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_stop_count",
		Help: "Number of actor stops",
	}, []string{"actor_type"})

	m.actorRestartCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_restart_count",
		Help: "Number of actor restarts",
	}, []string{"actor_type"})

	m.actorPassivateCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_passivate_count",
		Help: "Number of actor passivations",
	}, []string{"actor_type"})

	m.actorActivateCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_activate_count",
		Help: "Number of actor activations",
	}, []string{"actor_type"})

	m.actorErrorCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_error_count",
		Help: "Number of actor errors",
	}, []string{"actor_type", "error_type"})

	m.actorLifetime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "actor_lifetime_seconds",
		Help:    "Lifetime of actors in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 10, 6), // 1s to 1000000s
	}, []string{"actor_type"})

	// Supervision metrics
	m.supervisionRestartCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_supervision_restart_count",
		Help: "Number of actor restarts by supervisors",
	}, []string{"strategy"})

	m.supervisionStopCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_supervision_stop_count",
		Help: "Number of actor stops by supervisors",
	}, []string{"strategy"})

	m.supervisionErrorCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_supervision_error_count",
		Help: "Number of errors handled by supervisors",
	}, []string{"strategy", "error_type"})

	// Circuit breaker metrics
	m.circuitBreakerOpenCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_circuit_breaker_open_count",
		Help: "Number of times circuit breakers opened",
	}, []string{"name"})

	m.circuitBreakerCloseCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_circuit_breaker_close_count",
		Help: "Number of times circuit breakers closed",
	}, []string{"name"})

	m.circuitBreakerRejectCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "actor_circuit_breaker_reject_count",
		Help: "Number of requests rejected by circuit breakers",
	}, []string{"name"})

	m.circuitBreakerState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "actor_circuit_breaker_state",
		Help: "State of circuit breakers (0=closed, 1=open, 2=half-open)",
	}, []string{"name"})

	// Resource metrics
	m.memoryUsage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "actor_memory_usage_bytes",
		Help: "Memory usage of actors in bytes",
	}, []string{"actor_type"})

	m.cpuUsage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "actor_cpu_usage_percent",
		Help: "CPU usage of actors in percent",
	}, []string{"actor_type"})
}

// registerMetrics registers all metrics with the registry
func (m *ActorMetrics) registerMetrics() {
	// Actor counts
	m.registry.MustRegister(m.deviceActorCount)
	m.registry.MustRegister(m.twinActorCount)
	m.registry.MustRegister(m.roomActorCount)
	m.registry.MustRegister(m.pipelineActorCount)
	m.registry.MustRegister(m.totalActorCount)

	// Message metrics
	m.registry.MustRegister(m.messageCount)
	m.registry.MustRegister(m.messageLatency)
	m.registry.MustRegister(m.messageErrorCount)
	m.registry.MustRegister(m.messageSize)
	m.registry.MustRegister(m.messageQueueSize)
	m.registry.MustRegister(m.messageProcessTime)

	// Actor lifecycle metrics
	m.registry.MustRegister(m.actorStartCount)
	m.registry.MustRegister(m.actorStopCount)
	m.registry.MustRegister(m.actorRestartCount)
	m.registry.MustRegister(m.actorPassivateCount)
	m.registry.MustRegister(m.actorActivateCount)
	m.registry.MustRegister(m.actorErrorCount)
	m.registry.MustRegister(m.actorLifetime)

	// Supervision metrics
	m.registry.MustRegister(m.supervisionRestartCount)
	m.registry.MustRegister(m.supervisionStopCount)
	m.registry.MustRegister(m.supervisionErrorCount)

	// Circuit breaker metrics
	m.registry.MustRegister(m.circuitBreakerOpenCount)
	m.registry.MustRegister(m.circuitBreakerCloseCount)
	m.registry.MustRegister(m.circuitBreakerRejectCount)
	m.registry.MustRegister(m.circuitBreakerState)

	// Resource metrics
	m.registry.MustRegister(m.memoryUsage)
	m.registry.MustRegister(m.cpuUsage)
}

// Registry returns the Prometheus registry
func (m *ActorMetrics) Registry() *prometheus.Registry {
	return m.registry
}

// RecordMessageReceived records a message received by an actor
func (m *ActorMetrics) RecordMessageReceived(actorType, messageType string, size int) {
	m.messageCount.WithLabelValues(actorType, messageType).Inc()
	m.messageSize.WithLabelValues(actorType, messageType).Observe(float64(size))
}

// RecordMessageProcessed records a message processed by an actor
func (m *ActorMetrics) RecordMessageProcessed(actorType, messageType string, duration time.Duration) {
	m.messageProcessTime.WithLabelValues(actorType, messageType).Observe(duration.Seconds())
}

// RecordMessageError records a message processing error
func (m *ActorMetrics) RecordMessageError(actorType, messageType, errorType string) {
	m.messageErrorCount.WithLabelValues(actorType, messageType, errorType).Inc()
}

// RecordActorStarted records an actor start
func (m *ActorMetrics) RecordActorStarted(actorType string) {
	m.actorStartCount.WithLabelValues(actorType).Inc()
	m.updateActorCounts(actorType, 1)
}

// RecordActorStopped records an actor stop
func (m *ActorMetrics) RecordActorStopped(actorType string, lifetime time.Duration) {
	m.actorStopCount.WithLabelValues(actorType).Inc()
	m.actorLifetime.WithLabelValues(actorType).Observe(lifetime.Seconds())
	m.updateActorCounts(actorType, -1)
}

// RecordActorRestarted records an actor restart
func (m *ActorMetrics) RecordActorRestarted(actorType string) {
	m.actorRestartCount.WithLabelValues(actorType).Inc()
}

// RecordActorPassivated records an actor passivation
func (m *ActorMetrics) RecordActorPassivated(actorType string) {
	m.actorPassivateCount.WithLabelValues(actorType).Inc()
	m.updateActorCounts(actorType, -1)
}

// RecordActorActivated records an actor activation
func (m *ActorMetrics) RecordActorActivated(actorType string) {
	m.actorActivateCount.WithLabelValues(actorType).Inc()
	m.updateActorCounts(actorType, 1)
}

// RecordActorError records an actor error
func (m *ActorMetrics) RecordActorError(actorType, errorType string) {
	m.actorErrorCount.WithLabelValues(actorType, errorType).Inc()
}

// RecordSupervisionRestart records a supervision restart
func (m *ActorMetrics) RecordSupervisionRestart(strategy string) {
	m.supervisionRestartCount.WithLabelValues(strategy).Inc()
}

// RecordSupervisionStop records a supervision stop
func (m *ActorMetrics) RecordSupervisionStop(strategy string) {
	m.supervisionStopCount.WithLabelValues(strategy).Inc()
}

// RecordSupervisionError records a supervision error
func (m *ActorMetrics) RecordSupervisionError(strategy, errorType string) {
	m.supervisionErrorCount.WithLabelValues(strategy, errorType).Inc()
}

// RecordCircuitBreakerOpen records a circuit breaker opening
func (m *ActorMetrics) RecordCircuitBreakerOpen(name string) {
	m.circuitBreakerOpenCount.WithLabelValues(name).Inc()
	m.circuitBreakerState.WithLabelValues(name).Set(1) // 1 = open
}

// RecordCircuitBreakerClose records a circuit breaker closing
func (m *ActorMetrics) RecordCircuitBreakerClose(name string) {
	m.circuitBreakerCloseCount.WithLabelValues(name).Inc()
	m.circuitBreakerState.WithLabelValues(name).Set(0) // 0 = closed
}

// RecordCircuitBreakerHalfOpen records a circuit breaker half-opening
func (m *ActorMetrics) RecordCircuitBreakerHalfOpen(name string) {
	m.circuitBreakerState.WithLabelValues(name).Set(2) // 2 = half-open
}

// RecordCircuitBreakerReject records a circuit breaker rejecting a request
func (m *ActorMetrics) RecordCircuitBreakerReject(name string) {
	m.circuitBreakerRejectCount.WithLabelValues(name).Inc()
}

// RecordMemoryUsage records memory usage for an actor type
func (m *ActorMetrics) RecordMemoryUsage(actorType string, bytes float64) {
	m.memoryUsage.WithLabelValues(actorType).Set(bytes)
}

// RecordCPUUsage records CPU usage for an actor type
func (m *ActorMetrics) RecordCPUUsage(actorType string, percent float64) {
	m.cpuUsage.WithLabelValues(actorType).Set(percent)
}

// SetMessageQueueSize sets the message queue size for an actor
func (m *ActorMetrics) SetMessageQueueSize(actorType, actorID string, size float64) {
	m.messageQueueSize.WithLabelValues(actorType, actorID).Set(size)
}

// updateActorCounts updates the actor counts
func (m *ActorMetrics) updateActorCounts(actorType string, delta float64) {
	switch actorType {
	case "device":
		m.deviceActorCount.Add(delta)
	case "twin":
		m.twinActorCount.Add(delta)
	case "room":
		m.roomActorCount.Add(delta)
	case "pipeline":
		m.pipelineActorCount.Add(delta)
	}

	m.totalActorCount.Add(delta)
}

// Reset resets all metrics
func (m *ActorMetrics) Reset() {
	// Actor counts
	m.deviceActorCount.Set(0)
	m.twinActorCount.Set(0)
	m.roomActorCount.Set(0)
	m.pipelineActorCount.Set(0)
	m.totalActorCount.Set(0)

	// We can't reset counters and histograms in Prometheus
	// They are designed to only increase
	// For testing, you would need to create a new registry
}

// StartMetricsCollection starts collecting metrics
func (m *ActorMetrics) StartMetricsCollection(actorSystem *ActorSystem) {
	go m.collectMetrics(actorSystem)
}

// collectMetrics periodically collects metrics from the actor system
func (m *ActorMetrics) collectMetrics(actorSystem *ActorSystem) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Update actor counts
		deviceCount := float64(actorSystem.GetDeviceActorCount())
		twinCount := float64(actorSystem.GetTwinActorCount())
		roomCount := float64(actorSystem.GetRoomActorCount())
		pipelineCount := float64(actorSystem.GetPipelineActorCount())

		m.deviceActorCount.Set(deviceCount)
		m.twinActorCount.Set(twinCount)
		m.roomActorCount.Set(roomCount)
		m.pipelineActorCount.Set(pipelineCount)
		m.totalActorCount.Set(deviceCount + twinCount + roomCount + pipelineCount)

		// Update passivation metrics
		passiveCount := float64(actorSystem.passivationManager.GetActiveActorCount())
		m.logger.Debug("Collecting metrics", "active_actors", passiveCount)
	}
}
