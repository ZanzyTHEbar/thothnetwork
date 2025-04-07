package concurrent

import (
	"sync"
)

// HashMap is a thread-safe concurrent hash map implementation
type HashMap struct {
	shards     []*mapShard
	shardCount int
	shardMask  int
}

// mapShard represents a single shard of the concurrent hash map
type mapShard struct {
	items map[string]interface{}
	mu    sync.RWMutex
}

// NewHashMap creates a new concurrent hash map with the specified number of shards
// The number of shards should be a power of 2 for efficient hashing
func NewHashMap(shardCount int) *HashMap {
	// Ensure shard count is a power of 2
	if shardCount <= 0 || (shardCount&(shardCount-1)) != 0 {
		// Default to 16 shards if not a power of 2
		shardCount = 16
	}

	shards := make([]*mapShard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = &mapShard{
			items: make(map[string]interface{}),
		}
	}

	return &HashMap{
		shards:     shards,
		shardCount: shardCount,
		shardMask:  shardCount - 1,
	}
}

// getShard returns the shard for the given key
func (m *HashMap) getShard(key string) *mapShard {
	// Simple hash function for string keys
	hash := 0
	for i := 0; i < len(key); i++ {
		hash = 31*hash + int(key[i])
	}
	return m.shards[hash&m.shardMask]
}

// Get retrieves a value from the map
func (m *HashMap) Get(key string) (interface{}, bool) {
	shard := m.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	val, ok := shard.items[key]
	return val, ok
}

// Put adds or updates a value in the map
func (m *HashMap) Put(key string, value interface{}) {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.items[key] = value
}

// Remove deletes a key from the map
func (m *HashMap) Remove(key string) {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	delete(shard.items, key)
}

// Contains checks if a key exists in the map
func (m *HashMap) Contains(key string) bool {
	shard := m.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	_, ok := shard.items[key]
	return ok
}

// Size returns the total number of items in the map
func (m *HashMap) Size() int {
	count := 0
	for i := 0; i < m.shardCount; i++ {
		shard := m.shards[i]
		shard.mu.RLock()
		count += len(shard.items)
		shard.mu.RUnlock()
	}
	return count
}

// Keys returns all keys in the map
func (m *HashMap) Keys() []string {
	keys := make([]string, 0)
	for i := 0; i < m.shardCount; i++ {
		shard := m.shards[i]
		shard.mu.RLock()
		for k := range shard.items {
			keys = append(keys, k)
		}
		shard.mu.RUnlock()
	}
	return keys
}

// ForEach executes the provided function for each key-value pair in the map
func (m *HashMap) ForEach(fn func(key string, value interface{}) bool) {
	for i := 0; i < m.shardCount; i++ {
		shard := m.shards[i]
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
func (m *HashMap) Clear() {
	for i := 0; i < m.shardCount; i++ {
		shard := m.shards[i]
		shard.mu.Lock()
		shard.items = make(map[string]interface{})
		shard.mu.Unlock()
	}
}
