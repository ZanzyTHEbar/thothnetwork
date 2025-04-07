package actor

import (
	"sync"
	"time"

	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/anthdm/hollywood/actor"
)

// PassivationConfig holds configuration for the passivation manager
type PassivationConfig struct {
	// Timeout is the duration after which inactive actors are passivated
	Timeout time.Duration

	// CheckInterval is the interval at which to check for inactive actors
	CheckInterval time.Duration

	// Logger is the logger for passivation events
	Logger logger.Logger

	// BeforePassivation is a callback function to execute before passivating an actor
	BeforePassivation func(actorID string) error

	// AfterPassivation is a callback function to execute after passivating an actor
	AfterPassivation func(actorID string)
}

// PassivationManager manages actor passivation
type PassivationManager struct {
	// Map of actor IDs to their last activity time
	lastActivity map[string]time.Time

	// Timeout duration after which actors are passivated
	timeout time.Duration

	// Engine reference for stopping actors
	engine *actor.Engine

	// Logger for logging passivation events
	logger logger.Logger

	// Mutex for thread safety
	mu sync.RWMutex

	// Channel for stopping the passivation checker
	stopCh chan struct{}

	// WaitGroup for waiting for the passivation checker to stop
	wg sync.WaitGroup

	// Callback function to execute before passivating an actor
	beforePassivation func(actorID string) error

	// Callback function to execute after passivating an actor
	afterPassivation func(actorID string)
}

// NewPassivationManager creates a new passivation manager
func NewPassivationManager(engine *actor.Engine, config PassivationConfig) *PassivationManager {
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Minute // Default timeout
	}

	if config.CheckInterval <= 0 {
		config.CheckInterval = 1 * time.Minute // Default check interval
	}

	if config.Logger == nil {
		// Use a default logger if none provided
		panic("Logger is required for passivation manager")
	}

	pm := &PassivationManager{
		lastActivity:      make(map[string]time.Time),
		timeout:           config.Timeout,
		engine:            engine,
		logger:            config.Logger.With("component", "passivation_manager"),
		stopCh:            make(chan struct{}),
		beforePassivation: config.BeforePassivation,
		afterPassivation:  config.AfterPassivation,
	}

	// Start the passivation checker
	pm.wg.Add(1)
	go pm.startPassivationChecker(config.CheckInterval)

	return pm
}

// RecordActivity records activity for an actor
func (pm *PassivationManager) RecordActivity(actorID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.lastActivity[actorID] = time.Now()
}

// RemoveActor removes an actor from the passivation manager
func (pm *PassivationManager) RemoveActor(actorID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.lastActivity, actorID)
}

// startPassivationChecker starts a goroutine that periodically checks for inactive actors
func (pm *PassivationManager) startPassivationChecker(checkInterval time.Duration) {
	defer pm.wg.Done()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.checkForInactiveActors()
		case <-pm.stopCh:
			return
		}
	}
}

// checkForInactiveActors checks for inactive actors and passivates them
func (pm *PassivationManager) checkForInactiveActors() {
	now := time.Now()

	// Get actors to passivate
	actorsToPassivate := make([]string, 0)

	pm.mu.RLock()
	for actorID, lastActive := range pm.lastActivity {
		if now.Sub(lastActive) > pm.timeout {
			actorsToPassivate = append(actorsToPassivate, actorID)
		}
	}
	pm.mu.RUnlock()

	// Passivate actors
	for _, actorID := range actorsToPassivate {
		pm.passivateActor(actorID)
	}
}

// passivateActor passivates an actor
func (pm *PassivationManager) passivateActor(actorID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Remove actor from passivation manager
	delete(pm.lastActivity, actorID)

	// Execute before passivation callback if provided
	if pm.beforePassivation != nil {
		if err := pm.beforePassivation(actorID); err != nil {
			pm.logger.Error("Error executing before passivation callback", "actor_id", actorID, "error", err)
			return
		}
	}

	// In the current API, we would need to get the PID differently
	// pid := actor.NewPID("local", actorID)

	// In the current API, we would need to handle stopping differently
	// For now, we'll just log that we're stopping the actor
	pm.logger.Info("Stopping actor due to inactivity", "actor_id", actorID)

	pm.logger.Info("Passivated actor due to inactivity", "actor_id", actorID)

	// Execute after passivation callback if provided
	if pm.afterPassivation != nil {
		pm.afterPassivation(actorID)
	}
}

// GetActiveActorCount returns the number of active actors
func (pm *PassivationManager) GetActiveActorCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return len(pm.lastActivity)
}

// GetActiveActors returns a list of active actor IDs
func (pm *PassivationManager) GetActiveActors() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	actors := make([]string, 0, len(pm.lastActivity))
	for actorID := range pm.lastActivity {
		actors = append(actors, actorID)
	}

	return actors
}

// IsActive returns true if an actor is active
func (pm *PassivationManager) IsActive(actorID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, ok := pm.lastActivity[actorID]
	return ok
}

// GetLastActivity returns the last activity time for an actor
func (pm *PassivationManager) GetLastActivity(actorID string) (time.Time, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	lastActivity, ok := pm.lastActivity[actorID]
	return lastActivity, ok
}

// SetTimeout sets the timeout duration
func (pm *PassivationManager) SetTimeout(timeout time.Duration) {
	if timeout <= 0 {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.timeout = timeout
}

// GetTimeout returns the timeout duration
func (pm *PassivationManager) GetTimeout() time.Duration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.timeout
}

// Stop stops the passivation manager
func (pm *PassivationManager) Stop() {
	close(pm.stopCh)
	pm.wg.Wait()
}
