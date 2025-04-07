package actor

import (
	"time"

	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/anthdm/hollywood/actor"
)

// SupervisionStrategy defines the strategy for handling actor failures
type SupervisionStrategy int

const (
	// OneForOneStrategy restarts only the failed actor
	OneForOneStrategy SupervisionStrategy = iota
	// AllForOneStrategy restarts all child actors when one fails
	AllForOneStrategy
	// ExponentialBackoffStrategy uses exponential backoff for retries
	ExponentialBackoffStrategy
)

// Supervisor manages actor supervision
type Supervisor struct {
	strategy       SupervisionStrategy
	maxRetries     int
	withinDuration time.Duration
	logger         logger.Logger
	retryCount     map[string]int
	lastFailure    map[string]time.Time
}

// SupervisorConfig holds configuration for a supervisor
type SupervisorConfig struct {
	Strategy       SupervisionStrategy
	MaxRetries     int
	WithinDuration time.Duration
	Logger         logger.Logger
}

// NewSupervisor creates a new supervisor with the given configuration
func NewSupervisor(config SupervisorConfig) *Supervisor {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 10
	}
	if config.WithinDuration <= 0 {
		config.WithinDuration = 1 * time.Minute
	}

	return &Supervisor{
		strategy:       config.Strategy,
		maxRetries:     config.MaxRetries,
		withinDuration: config.WithinDuration,
		logger:         config.Logger,
		retryCount:     make(map[string]int),
		lastFailure:    make(map[string]time.Time),
	}
}

// HandleFailure handles actor failures based on the supervision strategy
func (s *Supervisor) HandleFailure(ctx *actor.Context, child *actor.PID, err error) {
	actorID := child.ID

	s.logger.Error("Actor failed", "actor_id", actorID, "error", err)

	// Check if we should reset retry count
	if lastFailure, ok := s.lastFailure[actorID]; ok {
		if time.Since(lastFailure) > s.withinDuration {
			s.retryCount[actorID] = 0
		}
	}

	// Update failure time
	s.lastFailure[actorID] = time.Now()

	// Increment retry count
	s.retryCount[actorID]++

	// Check if we've exceeded max retries
	if s.retryCount[actorID] > s.maxRetries {
		s.logger.Error("Actor exceeded max retries, stopping", "actor_id", actorID, "retries", s.retryCount[actorID])
		ctx.Stop(child)
		return
	}

	// Apply supervision strategy
	switch s.strategy {
	case OneForOneStrategy:
		// Restart only the failed actor
		s.logger.Info("Restarting actor (OneForOne)", "actor_id", actorID, "retry", s.retryCount[actorID])
		ctx.Restart(child)

	case AllForOneStrategy:
		// Restart all child actors
		s.logger.Info("Restarting all actors (AllForOne)", "actor_id", actorID, "retry", s.retryCount[actorID])
		// In a real implementation, we would need to track all children
		// For now, just restart the failed actor
		ctx.Restart(child)

	case ExponentialBackoffStrategy:
		// Use exponential backoff for retries
		backoff := time.Duration(1<<uint(s.retryCount[actorID]-1)) * time.Second
		if backoff > 1*time.Minute {
			backoff = 1 * time.Minute
		}
		s.logger.Info("Restarting actor with backoff", "actor_id", actorID, "retry", s.retryCount[actorID], "backoff", backoff)
		time.AfterFunc(backoff, func() {
			ctx.Restart(child)
		})
	}
}

// CreateOneForOneSupervisor creates a supervisor with OneForOne strategy
func CreateOneForOneSupervisor(maxRetries int, withinDuration time.Duration, logger logger.Logger) *Supervisor {
	return NewSupervisor(SupervisorConfig{
		Strategy:       OneForOneStrategy,
		MaxRetries:     maxRetries,
		WithinDuration: withinDuration,
		Logger:         logger,
	})
}

// CreateAllForOneSupervisor creates a supervisor with AllForOne strategy
func CreateAllForOneSupervisor(maxRetries int, withinDuration time.Duration, logger logger.Logger) *Supervisor {
	return NewSupervisor(SupervisorConfig{
		Strategy:       AllForOneStrategy,
		MaxRetries:     maxRetries,
		WithinDuration: withinDuration,
		Logger:         logger,
	})
}

// CreateExponentialBackoffSupervisor creates a supervisor with ExponentialBackoff strategy
func CreateExponentialBackoffSupervisor(maxRetries int, withinDuration time.Duration, logger logger.Logger) *Supervisor {
	return NewSupervisor(SupervisorConfig{
		Strategy:       ExponentialBackoffStrategy,
		MaxRetries:     maxRetries,
		WithinDuration: withinDuration,
		Logger:         logger,
	})
}
