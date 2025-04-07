package concurrent

import (
	"github.com/anthdm/hollywood/actor"
)

// ActorMap is a specialized concurrent map for storing actor references
type ActorMap struct {
	syncMap *SyncMap
}

// NewActorMap creates a new concurrent map for actor references
func NewActorMap(numShards int) *ActorMap {
	return &ActorMap{
		syncMap: NewSyncMap(numShards),
	}
}

// Get retrieves an actor PID by ID
func (m *ActorMap) Get(actorID string) (*actor.PID, bool) {
	val, ok := m.syncMap.Get(actorID)
	if !ok {
		return nil, false
	}
	return val.(*actor.PID), true
}

// Set adds or updates an actor PID
func (m *ActorMap) Set(actorID string, pid *actor.PID) {
	m.syncMap.Set(actorID, pid)
}

// Delete removes an actor PID
func (m *ActorMap) Delete(actorID string) {
	m.syncMap.Delete(actorID)
}

// Has checks if an actor ID exists
func (m *ActorMap) Has(actorID string) bool {
	return m.syncMap.Has(actorID)
}

// Len returns the total number of actors
func (m *ActorMap) Len() int {
	return m.syncMap.Len()
}

// Keys returns all actor IDs
func (m *ActorMap) Keys() []string {
	keys := m.syncMap.Keys()
	result := make([]string, len(keys))
	for i, k := range keys {
		result[i] = k.(string)
	}
	return result
}

// ForEach executes the provided function for each actor
func (m *ActorMap) ForEach(fn func(actorID string, pid *actor.PID) bool) {
	m.syncMap.ForEach(func(key, value any) bool {
		return fn(key.(string), value.(*actor.PID))
	})
}

// Clear removes all actors
func (m *ActorMap) Clear() {
	m.syncMap.Clear()
}
