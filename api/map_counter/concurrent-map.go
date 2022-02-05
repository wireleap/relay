// Copyright (c) 2022 Wireleap

package map_counter

import (
	"github.com/wireleap/relay/api/synccounters"

	"sync"
)

var SHARD_COUNT = 32

// A "thread" safe map of type string:Anything.
// To avoid lock bottlenecks this map is dived to several (SHARD_COUNT) map shards.
type concurrentMap []*ConcurrentMapShared

// A "thread" safe string to anything map.
type ConcurrentMapShared struct {
	items        map[string]*synccounters.ContractCounter
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// Creates a new concurrent map.
func New() concurrentMap {
	m := make(concurrentMap, SHARD_COUNT)
	for i := 0; i < SHARD_COUNT; i++ {
		m[i] = &ConcurrentMapShared{items: make(map[string]*synccounters.ContractCounter)}
	}
	return m
}

// GetShard returns shard under given key
func (m concurrentMap) GetShard(key string) *ConcurrentMapShared {
	return m[uint(fnv32(key))%uint(SHARD_COUNT)]
}

// Callback to return new element to be inserted into the map
// It is called while lock is held, therefore it MUST NOT
// try to access other keys in same map, as it can lead to deadlock since
// Go sync.RWLock is not reentrant
type UpsertCb func(exist bool, valueInMap *uint64, newValue *uint64) *uint64

// Sets the given value under the specified key if no value was associated with it.
func (m concurrentMap) SetIfAbsent(key string, value *synccounters.ContractCounter) bool {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	_, ok := shard.items[key]
	if !ok {
		shard.items[key] = value
	}
	shard.Unlock()
	return !ok
}

// Get retrieves an element from map under given key.
func (m concurrentMap) Get(key string) (*synccounters.ContractCounter, bool) {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	// Get item from shard.
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type Tuple struct {
	Key string
	Val *synccounters.ContractCounter
}

// Iterator callback,called for every key,value found in
// maps. RLock is held for all calls for a given shard
// therefore callback sess consistent view of a shard,
// but not across the shards
type IterCb func(key string, v *synccounters.ContractCounter) bool

// Callback based iterator, cheapest way to read
// all elements in a map.
func (m concurrentMap) IterCb(fn IterCb) bool {
	for idx := range m {
		shard := (m)[idx]
		shard.RLock()
		bk := false
		for key, value := range shard.items {
			if bk = !fn(key, value); bk {
				shard.RUnlock()
				return false
			}
		}
		shard.RUnlock()
	}
	return true
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

// Custom Mini API
type CMap struct{ m concurrentMap }

func NewCMap() Map {
	c := new(CMap)
	c.m = New()
	return c
}

func (c *CMap) GetOrInit(key string) (value *synccounters.ConnCounter) {
	ctc := synccounters.NewContractCounter()
	for { // This loop is due to quantum physics
		if c.m.SetIfAbsent(key, ctc) {
			return ctc.NewChild()
		} else if v, ok := c.m.Get(key); ok {
			return v.NewChild()
		}
	}
}

func (c *CMap) Range(f func(key string, value *synccounters.ContractCounter) bool) bool {
	return c.m.IterCb(f)
}

func (c *CMap) Reset() (map[string]uint64, bool) {
	return resetMap(c)
}
