package health

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
)

// Status represents the status of a component
type Status string

const (
	// StatusUp means the component is up and running
	StatusUp Status = "up"
	// StatusDown means the component is down
	StatusDown Status = "down"
	// StatusDegraded means the component is running but with issues
	StatusDegraded Status = "degraded"
	// StatusUnknown means the status of the component is unknown
	StatusUnknown Status = "unknown"
)

// Check represents a health check
type Check struct {
	// Name is the name of the check
	Name string `json:"name"`

	// Status is the status of the check
	Status Status `json:"status"`

	// Message is a message describing the status
	Message string `json:"message,omitempty"`

	// LastChecked is the time the check was last performed
	LastChecked time.Time `json:"last_checked"`

	// Metadata is additional information about the check
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CheckFunc is a function that performs a health check
type CheckFunc func(ctx context.Context) Check

// Checker manages health checks
type HealthChecker struct {
	checks map[string]CheckFunc
	cache  map[string]Check
	mu     sync.RWMutex
	logger logger.Logger
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(logger logger.Logger) *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]CheckFunc),
		cache:  make(map[string]Check),
		logger: logger.With("component", "health_checker"),
	}
}

// RegisterCheck registers a health check
func (h *HealthChecker) RegisterCheck(name string, check CheckFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.checks[name] = check
	h.logger.Debug("Registered health check", "name", name)
}

// UnregisterCheck unregisters a health check
func (h *HealthChecker) UnregisterCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.checks, name)
	delete(h.cache, name)
	h.logger.Debug("Unregistered health check", "name", name)
}

// RunChecks runs all registered health checks
func (h *HealthChecker) RunChecks(ctx context.Context) map[string]Check {
	h.mu.Lock()
	defer h.mu.Unlock()

	results := make(map[string]Check)

	for name, check := range h.checks {
		h.logger.Debug("Running health check", "name", name)

		// Run the check
		result := check(ctx)

		// Update the cache
		h.cache[name] = result

		// Add to results
		results[name] = result
	}

	return results
}

// GetStatus gets the overall status of the system
func (h *HealthChecker) GetStatus(ctx context.Context) Status {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// If there are no checks, return unknown
	if len(h.checks) == 0 {
		return StatusUnknown
	}

	// Run checks if cache is empty
	if len(h.cache) == 0 {
		h.mu.RUnlock()
		h.RunChecks(ctx)
		h.mu.RLock()
	}

	// Check if any check is down
	for _, check := range h.cache {
		if check.Status == StatusDown {
			return StatusDown
		}
	}

	// Check if any check is degraded
	for _, check := range h.cache {
		if check.Status == StatusDegraded {
			return StatusDegraded
		}
	}

	// Check if any check is unknown
	for _, check := range h.cache {
		if check.Status == StatusUnknown {
			return StatusUnknown
		}
	}

	// All checks are up
	return StatusUp
}

// GetChecks gets all health checks
func (h *HealthChecker) GetChecks(ctx context.Context, forceRun bool) map[string]Check {
	if forceRun {
		return h.RunChecks(ctx)
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// If cache is empty, run checks
	if len(h.cache) == 0 {
		h.mu.RUnlock()
		return h.RunChecks(ctx)
	}

	// Return a copy of the cache
	results := make(map[string]Check, len(h.cache))
	maps.Copy(results, h.cache)

	return results
}

// GetCheck gets a specific health check
func (h *HealthChecker) GetCheck(ctx context.Context, name string, forceRun bool) (Check, bool) {
	h.mu.RLock()

	// Check if the check exists
	check, ok := h.checks[name]
	if !ok {
		h.mu.RUnlock()
		return Check{}, false
	}

	// If not forcing a run, return the cached result if available
	if !forceRun {
		if cachedCheck, ok := h.cache[name]; ok {
			h.mu.RUnlock()
			return cachedCheck, true
		}
	}

	h.mu.RUnlock()

	// Run the check
	h.mu.Lock()
	defer h.mu.Unlock()

	result := check(ctx)
	h.cache[name] = result

	return result, true
}

// Response is the response from the health endpoint
type HealthResponse struct {
	// Status is the overall status of the system
	Status Status `json:"status"`

	// Checks are the individual health checks
	Checks map[string]Check `json:"checks,omitempty"`

	// Timestamp is the time the health check was performed
	Timestamp time.Time `json:"timestamp"`
}

// ServeHTTP serves the health check endpoint
func (h *HealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	forceRun := query.Get("force") == "true"
	includeChecks := query.Get("include_checks") == "true"

	// Get status
	status := h.GetStatus(r.Context())

	// Prepare response
	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
	}

	// Include checks if requested
	if includeChecks {
		response.Checks = h.GetChecks(r.Context(), forceRun)
	}

	// Set status code based on status
	switch status {
	case StatusUp:
		w.WriteHeader(http.StatusOK)
	case StatusDegraded:
		w.WriteHeader(http.StatusOK)
	case StatusDown:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// StartServer starts a health check server
func (h *HealthChecker) StartServer(addr string) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.Handle("/health", h)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("Health check server failed", "error", err)
		}
	}()

	h.logger.Info("Health check server started", "addr", addr)

	return server, nil
}

// StopServer stops the health check server
func (h *HealthChecker) StopServer(ctx context.Context, server *http.Server) error {
	h.logger.Info("Stopping health check server")
	return server.Shutdown(ctx)
}

// DatabaseCheck creates a health check for a database
func DatabaseCheck(name string, pingFunc func(context.Context) error) CheckFunc {
	return func(ctx context.Context) Check {
		start := time.Now()
		err := pingFunc(ctx)
		latency := time.Since(start)

		check := Check{
			Name:        name,
			LastChecked: time.Now(),
			Metadata: map[string]any{
				"latency_ms": latency.Milliseconds(),
			},
		}

		if err != nil {
			check.Status = StatusDown
			check.Message = err.Error()
		} else {
			check.Status = StatusUp
			check.Message = "Database is up"
		}

		return check
	}
}

// HTTPCheck creates a health check for an HTTP endpoint
func HTTPCheck(name, url string, timeout time.Duration) CheckFunc {
	return func(ctx context.Context) Check {
		client := &http.Client{
			Timeout: timeout,
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return Check{
				Name:        name,
				Status:      StatusDown,
				Message:     err.Error(),
				LastChecked: time.Now(),
			}
		}

		start := time.Now()
		resp, err := client.Do(req)
		latency := time.Since(start)

		check := Check{
			Name:        name,
			LastChecked: time.Now(),
			Metadata: map[string]any{
				"latency_ms": latency.Milliseconds(),
			},
		}

		if err != nil {
			check.Status = StatusDown
			check.Message = err.Error()
			return check
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			check.Status = StatusUp
			check.Message = "HTTP endpoint is up"
		} else if resp.StatusCode >= 500 {
			check.Status = StatusDown
			check.Message = fmt.Sprintf("HTTP endpoint returned status code %d", resp.StatusCode)
		} else {
			check.Status = StatusDegraded
			check.Message = fmt.Sprintf("HTTP endpoint returned status code %d", resp.StatusCode)
		}

		return check
	}
}

// MemoryCheck creates a health check for memory usage
func MemoryCheck(name string, warningThreshold, criticalThreshold float64, getUsageFunc func() (float64, error)) CheckFunc {
	return func(ctx context.Context) Check {
		usage, err := getUsageFunc()

		check := Check{
			Name:        name,
			LastChecked: time.Now(),
			Metadata: map[string]any{
				"usage_percent": usage,
			},
		}

		if err != nil {
			check.Status = StatusUnknown
			check.Message = err.Error()
		} else if usage >= criticalThreshold {
			check.Status = StatusDown
			check.Message = fmt.Sprintf("Memory usage is critical: %.2f%%", usage)
		} else if usage >= warningThreshold {
			check.Status = StatusDegraded
			check.Message = fmt.Sprintf("Memory usage is high: %.2f%%", usage)
		} else {
			check.Status = StatusUp
			check.Message = fmt.Sprintf("Memory usage is normal: %.2f%%", usage)
		}

		return check
	}
}

// DiskCheck creates a health check for disk usage
func DiskCheck(name, path string, warningThreshold, criticalThreshold float64, getUsageFunc func(string) (float64, error)) CheckFunc {
	return func(ctx context.Context) Check {
		usage, err := getUsageFunc(path)

		check := Check{
			Name:        name,
			LastChecked: time.Now(),
			Metadata: map[string]any{
				"path":          path,
				"usage_percent": usage,
			},
		}

		if err != nil {
			check.Status = StatusUnknown
			check.Message = err.Error()
		} else if usage >= criticalThreshold {
			check.Status = StatusDown
			check.Message = fmt.Sprintf("Disk usage is critical: %.2f%%", usage)
		} else if usage >= warningThreshold {
			check.Status = StatusDegraded
			check.Message = fmt.Sprintf("Disk usage is high: %.2f%%", usage)
		} else {
			check.Status = StatusUp
			check.Message = fmt.Sprintf("Disk usage is normal: %.2f%%", usage)
		}

		return check
	}
}

// CPUCheck creates a health check for CPU usage
func CPUCheck(name string, warningThreshold, criticalThreshold float64, getUsageFunc func() (float64, error)) CheckFunc {
	return func(ctx context.Context) Check {
		usage, err := getUsageFunc()

		check := Check{
			Name:        name,
			LastChecked: time.Now(),
			Metadata: map[string]any{
				"usage_percent": usage,
			},
		}

		if err != nil {
			check.Status = StatusUnknown
			check.Message = err.Error()
		} else if usage >= criticalThreshold {
			check.Status = StatusDown
			check.Message = fmt.Sprintf("CPU usage is critical: %.2f%%", usage)
		} else if usage >= warningThreshold {
			check.Status = StatusDegraded
			check.Message = fmt.Sprintf("CPU usage is high: %.2f%%", usage)
		} else {
			check.Status = StatusUp
			check.Message = fmt.Sprintf("CPU usage is normal: %.2f%%", usage)
		}

		return check
	}
}

// CompositeCheck creates a health check that combines multiple checks
func CompositeCheck(name string, checks ...CheckFunc) CheckFunc {
	return func(ctx context.Context) Check {
		results := make([]Check, 0, len(checks))

		for _, check := range checks {
			results = append(results, check(ctx))
		}

		// Determine overall status
		status := StatusUp
		message := "All systems operational"

		for _, result := range results {
			if result.Status == StatusDown {
				status = StatusDown
				message = "One or more systems are down"
				break
			}
		}

		if status != StatusDown {
			for _, result := range results {
				if result.Status == StatusDegraded {
					status = StatusDegraded
					message = "One or more systems are degraded"
					break
				}
			}
		}

		if status != StatusDown && status != StatusDegraded {
			for _, result := range results {
				if result.Status == StatusUnknown {
					status = StatusUnknown
					message = "One or more systems have unknown status"
					break
				}
			}
		}

		return Check{
			Name:        name,
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
			Metadata: map[string]any{
				"checks": results,
			},
		}
	}
}
