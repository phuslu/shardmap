package shardmap

import (
	"runtime"
	"sync"
	"unsafe"
)

// Map is a hashmap. Like map[comparable]any, but sharded and thread-safe.
//
// The zero value is not safe for use; use New.
type Map[K comparable, V any] struct {
	mus    []syncRWMutex
	shards []shard[K, V]
	ksize  int
	cap    int
}

type syncRWMutex struct {
	sync.RWMutex
	_ [64 - unsafe.Sizeof(sync.RWMutex{})]byte // avoid false sharing
}

// New returns a new hashmap with the specified capacity.
func New[K comparable, V any](cap int) (m *Map[K, V]) {
	m = &Map[K, V]{cap: cap}

	n := 1
	for n < runtime.NumCPU()*16 {
		n *= 2
	}
	scap := m.cap / n
	m.mus = make([]syncRWMutex, n)
	m.shards = make([]shard[K, V], n)
	for i := 0; i < n; i++ {
		m.shards[i].init(scap)
	}

	var k K
	switch ((any)(k)).(type) {
	case string:
		m.ksize = 0
	default:
		m.ksize = int(unsafe.Sizeof(k))
	}

	return
}

// Clear out all values from map
func (m *Map[K, V]) Clear() {
	for i := 0; i < len(m.mus); i++ {
		m.mus[i].Lock()
		m.shards[i].init(m.cap / len(m.mus))
		m.mus[i].Unlock()
	}
}

// Set assigns a value to a key.
// Returns the previous value, or false when no value was assigned.
func (m *Map[K, V]) Set(key K, value V) (prev V, replaced bool) {
	var hash uint64
	if m.ksize == 0 {
		hash = wyhash_HashString(*(*string)(unsafe.Pointer(&key)), 0)
	} else {
		hash = wyhash_HashString(*(*string)(unsafe.Pointer(&struct {
			data unsafe.Pointer
			len  int
		}{unsafe.Pointer(&key), m.ksize})), 0)
	}
	shard := int(hash & uint64(len(m.mus)-1))
	m.mus[shard].Lock()
	prev, replaced = m.shards[shard].Set(hash, key, value)
	m.mus[shard].Unlock()
	return prev, replaced
}

// Get returns a value for a key.
// Returns false when no value has been assign for key.
func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	var hash uint64
	if m.ksize == 0 {
		hash = wyhash_HashString(*(*string)(unsafe.Pointer(&key)), 0)
	} else {
		hash = wyhash_HashString(*(*string)(unsafe.Pointer(&struct {
			data unsafe.Pointer
			len  int
		}{unsafe.Pointer(&key), m.ksize})), 0)
	}
	shard := int(hash & uint64(len(m.mus)-1))
	m.mus[shard].RLock()
	value, ok = m.shards[shard].Get(hash, key)
	m.mus[shard].RUnlock()
	return value, ok
}

// Delete deletes a value for a key.
// Returns the deleted value, or false when no value was assigned.
func (m *Map[K, V]) Delete(key K) (prev V, deleted bool) {
	var hash uint64
	if m.ksize == 0 {
		hash = wyhash_HashString(*(*string)(unsafe.Pointer(&key)), 0)
	} else {
		hash = wyhash_HashString(*(*string)(unsafe.Pointer(&struct {
			data unsafe.Pointer
			len  int
		}{unsafe.Pointer(&key), m.ksize})), 0)
	}
	shard := int(hash & uint64(len(m.mus)-1))
	m.mus[shard].Lock()
	prev, deleted = m.shards[shard].Delete(hash, key)
	m.mus[shard].Unlock()
	return prev, deleted
}

// Mutate atomically mutates m[k] by calling mutator.
//
// The mutator function is called with the old value (or its zero value) and
// whether it existed in the map and it returns the new value and whether it
// should be set in the map (true) or deleted from the map (false).
//
// It returns the change in size of the map as a result of the mutation, one of
// -1 (delete), 0 (change), or 1 (addition).
func (m *Map[K, V]) Mutate(key K, mutator func(oldValue V, oldValueExisted bool) (newValue V, keep bool)) (delta int) {
	var hash uint64
	if m.ksize == 0 {
		hash = wyhash_HashString(*(*string)(unsafe.Pointer(&key)), 0)
	} else {
		hash = wyhash_HashString(*(*string)(unsafe.Pointer(&struct {
			data unsafe.Pointer
			len  int
		}{unsafe.Pointer(&key), m.ksize})), 0)
	}
	shard := int(hash & uint64(len(m.mus)-1))
	m.mus[shard].Lock()
	defer m.mus[shard].Unlock()
	oldV, oldOK := m.shards[shard].Get(hash, key)
	newV, newOK := mutator(oldV, oldOK)
	if newOK {
		m.shards[shard].Set(hash, key, newV)
		if oldOK {
			return 0
		}
		return 1
	}
	m.shards[shard].Delete(hash, key)
	if oldOK {
		return -1
	}
	return 0
}

// Len returns the number of values in map.
func (m *Map[K, V]) Len() int {
	var n int
	for i := 0; i < len(m.mus); i++ {
		m.mus[i].Lock()
		n += m.shards[i].Len()
		m.mus[i].Unlock()
	}
	return n
}

// Range iterates overall all key/values.
// It's not safe to call or Set or Delete while ranging.
func (m *Map[K, V]) Range(iter func(key K, value V) bool) {
	var done bool
	for i := 0; i < len(m.mus); i++ {
		m.mus[i].RLock()
		m.shards[i].Range(func(key K, value V) bool {
			if !iter(key, value) {
				done = true
				return false
			}
			return true
		})
		m.mus[i].RUnlock()
		if done {
			break
		}
	}
}
