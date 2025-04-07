package concurrent

import (
	"sync"
)

// SyncMap provides a thread-safe map implementation with sharding for better performance
// under high concurrency scenarios.
type SyncMap struct {
	shards    []*syncMapShard
	numShards int
	mask      uint32
}

// syncMapShard represents a single shard of the concurrent map
type syncMapShard struct {
	items map[any]any
	mu    sync.RWMutex
}

// NewSyncMap creates a new concurrent map with the specified number of shards.
// The number of shards should be a power of 2 for efficient hashing.
func NewSyncMap(numShards int) *SyncMap {
	// Ensure numShards is a power of 2
	if numShards <= 0 || (numShards&(numShards-1)) != 0 {
		numShards = 16 // Default to 16 shards if not a power of 2
	}

	shards := make([]*syncMapShard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &syncMapShard{
			items: make(map[any]any),
		}
	}

	return &SyncMap{
		shards:    shards,
		numShards: numShards,
		mask:      uint32(numShards - 1),
	}
}

// getShard returns the shard for the given key
func (m *SyncMap) getShard(key any) *syncMapShard {
	hash := getHash(key)
	return m.shards[hash&m.mask]
}

// Get retrieves a value from the map
func (m *SyncMap) Get(key any) (any, bool) {
	shard := m.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	val, ok := shard.items[key]
	return val, ok
}

// Set adds or updates a value in the map
func (m *SyncMap) Set(key, value any) {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.items[key] = value
}

// Delete removes a key from the map
func (m *SyncMap) Delete(key any) {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	delete(shard.items, key)
}

// Has checks if a key exists in the map
func (m *SyncMap) Has(key any) bool {
	shard := m.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	_, ok := shard.items[key]
	return ok
}

// Len returns the total number of items in the map
func (m *SyncMap) Len() int {
	count := 0
	for _, shard := range m.shards {
		shard.mu.RLock()
		count += len(shard.items)
		shard.mu.RUnlock()
	}
	return count
}

// Keys returns all keys in the map
func (m *SyncMap) Keys() []any {
	keys := make([]any, 0)
	for _, shard := range m.shards {
		shard.mu.RLock()
		for k := range shard.items {
			keys = append(keys, k)
		}
		shard.mu.RUnlock()
	}
	return keys
}

// ForEach executes the provided function for each key-value pair in the map
func (m *SyncMap) ForEach(fn func(key, value any) bool) {
	for _, shard := range m.shards {
		shard.mu.RLock()
		for k, v := range shard.items {
			if !fn(k, v) {
				shard.mu.RUnlock()
				return
			}
		}
		shard.mu.RUnlock()
	}
}

// Clear removes all items from the map
func (m *SyncMap) Clear() {
	for _, shard := range m.shards {
		shard.mu.Lock()
		shard.items = make(map[any]any)
		shard.mu.Unlock()
	}
}

// getHash returns a hash for the given key
func getHash(key any) uint32 {
	switch k := key.(type) {
	case string:
		return stringHash(k)
	case int:
		return uint32(k)
	case int32:
		return uint32(k)
	case int64:
		return uint32(k)
	case uint32:
		return k
	case uint64:
		return uint32(k)
	default:
		// Use a simple hash for other types
		return 0
	}
}

// stringHash implements a simple but effective string hash function
func stringHash(s string) uint32 {
	h := uint32(2166136261)
	for _, c := range s {
		h ^= uint32(c)
		h *= 16777619
	}
	return h
}
