package actor

import (
	"sync"
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
	children       map[string]*actor.PID
	mu             sync.RWMutex
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
	// Ensure logger is provided
	if config.Logger == nil {
		panic("Logger is required for supervisor")
	}

	return &Supervisor{
		strategy:       config.Strategy,
		maxRetries:     config.MaxRetries,
		withinDuration: config.WithinDuration,
		logger:         config.Logger.With("component", "supervisor"),
		retryCount:     make(map[string]int),
		lastFailure:    make(map[string]time.Time),
		children:       make(map[string]*actor.PID),
	}
}

// RegisterChild registers a child actor with the supervisor
func (s *Supervisor) RegisterChild(actorID string, pid *actor.PID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.children[actorID] = pid
	s.logger.Debug("Registered child actor", "actor_id", actorID)
}

// UnregisterChild unregisters a child actor from the supervisor
func (s *Supervisor) UnregisterChild(actorID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.children, actorID)
	delete(s.retryCount, actorID)
	delete(s.lastFailure, actorID)
	s.logger.Debug("Unregistered child actor", "actor_id", actorID)
}

// HandleFailure handles actor failures based on the supervision strategy
func (s *Supervisor) HandleFailure(engine *actor.Engine, child *actor.PID, err error) {
	actorID := child.ID

	s.mu.Lock()
	defer s.mu.Unlock()

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
		// Stop the actor
		engine.Stop(child)

		// Remove from supervision
		delete(s.children, actorID)
		delete(s.retryCount, actorID)
		delete(s.lastFailure, actorID)

		return
	}

	// Apply supervision strategy
	switch s.strategy {
	case OneForOneStrategy:
		// Restart only the failed actor
		s.logger.Info("Restarting actor (OneForOne)", "actor_id", actorID, "retry", s.retryCount[actorID])
		// In the current API, we would need to handle restarting differently
		// For now, we'll just log that we're restarting
		s.logger.Info("Would restart actor", "actor_id", actorID)

	case AllForOneStrategy:
		// Restart all child actors
		s.logger.Info("Restarting all actors (AllForOne)", "actor_id", actorID, "retry", s.retryCount[actorID])
		// Now we can restart all children since we track them
		for id := range s.children {
			s.logger.Info("Would restart child actor", "actor_id", id)
			// In the current API, we would need to handle restarting differently
		}

	case ExponentialBackoffStrategy:
		// Use exponential backoff for retries
		backoff := time.Duration(1<<uint(s.retryCount[actorID]-1)) * time.Second
		if backoff > 1*time.Minute {
			backoff = 1 * time.Minute
		}
		s.logger.Info("Restarting actor with backoff", "actor_id", actorID, "retry", s.retryCount[actorID], "backoff", backoff)

		// Store actorID and child in local variables to avoid closure issues
		currentActorID := actorID

		time.AfterFunc(backoff, func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			// Check if actor is still registered
			if _, ok := s.children[currentActorID]; ok {
				// In the current API, we would need to handle restarting differently
				s.logger.Info("Would restart actor after backoff", "actor_id", currentActorID)
			}
		})
	}
}

// GetChildrenCount returns the number of children
func (s *Supervisor) GetChildrenCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.children)
}

// GetChildren returns a copy of the children map
func (s *Supervisor) GetChildren() map[string]*actor.PID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*actor.PID, len(s.children))
	for k, v := range s.children {
		result[k] = v
	}

	return result
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
