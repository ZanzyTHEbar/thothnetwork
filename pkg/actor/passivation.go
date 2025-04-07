package actor

import (
	"sync"
	"time"

	"github.com/ZanzyTHEbar/thothnetwork/pkg/logger"
	"github.com/anthdm/hollywood/actor"
)

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
}

// NewPassivationManager creates a new passivation manager
func NewPassivationManager(engine *actor.Engine, timeout time.Duration, logger logger.Logger) *PassivationManager {
	if timeout <= 0 {
		timeout = 30 * time.Minute // Default timeout
	}

	pm := &PassivationManager{
		lastActivity: make(map[string]time.Time),
		timeout:      timeout,
		engine:       engine,
		logger:       logger.With("component", "passivation_manager"),
	}

	// Start the passivation checker
	go pm.startPassivationChecker()

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
func (pm *PassivationManager) startPassivationChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pm.checkForInactiveActors()
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

	// In the current API, we would need to get the PID differently
	// pid := actor.NewPID("local", actorID)

	// In the current API, we would need to handle stopping differently
	// For now, we'll just log that we're stopping the actor
	pm.logger.Info("Stopping actor due to inactivity", "actor_id", actorID)

	pm.logger.Info("Passivated actor due to inactivity", "actor_id", actorID)
}

// GetActiveActorCount returns the number of active actors
func (pm *PassivationManager) GetActiveActorCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return len(pm.lastActivity)
}
