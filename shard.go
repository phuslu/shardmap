// Copyright 2019 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an ISC-style
// license that can be found in the LICENSE file.

package shardmap

const (
	loadFactor  = 0.85                      // must be above 50%
	dibBitSize  = 16                        // 0xFFFF
	hashBitSize = 64 - dibBitSize           // 0xFFFFFFFFFFFF
	maxHash     = ^uint64(0) >> dibBitSize  // max 28,147,497,671,0655
	maxDIB      = ^uint64(0) >> hashBitSize // max 65,535
)

type entry[K comparable, V any] struct {
	hdib  uint64 // bitfield { hash:48 dib:16 }
	key   K      // user key
	value V      // user value
}

// Map is a hashmap. Like map[comparable]any
type shard[K comparable, V any] struct {
	buckets  []entry[K, V]
	cap      int
	length   int
	mask     int
	growAt   int
	shrinkAt int
}

// New returns a new Map. Like map[comparable]any
func newshard[K comparable, V any](cap int) *shard[K, V] {
	m := new(shard[K, V])
	m.cap = cap
	sz := 8
	for sz < m.cap {
		sz *= 2
	}
	m.buckets = make([]entry[K, V], sz)
	m.mask = len(m.buckets) - 1
	m.growAt = int(float64(len(m.buckets)) * loadFactor)
	m.shrinkAt = int(float64(len(m.buckets)) * (1 - loadFactor))
	return m
}

func (m *shard[K, V]) resize(newCap int) {
	nmap := newshard[K, V](newCap)
	for i := 0; i < len(m.buckets); i++ {
		if int(m.buckets[i].hdib&maxDIB) > 0 {
			nmap.set(int(m.buckets[i].hdib>>dibBitSize), m.buckets[i].key, m.buckets[i].value)
		}
	}
	cap := m.cap
	*m = *nmap
	m.cap = cap
}

// Set assigns a value to a key.
// Returns the previous value, or false when no value was assigned.
func (m *shard[K, V]) Set(xxh uint64, key K, value V) (V, bool) {
	if len(m.buckets) == 0 {
		*m = *newshard[K, V](0)
	}
	if m.length >= m.growAt {
		m.resize(len(m.buckets) * 2)
	}
	return m.set(int(xxh>>dibBitSize), key, value)
}

func (m *shard[K, V]) set(hash int, key K, value V) (prev V, ok bool) {
	e := entry[K, V]{uint64(hash)<<dibBitSize | uint64(1)&maxDIB, key, value}
	i := int(e.hdib>>dibBitSize) & m.mask
	for {
		if int(m.buckets[i].hdib&maxDIB) == 0 {
			m.buckets[i] = e
			m.length++
			return
		}
		if int(e.hdib>>dibBitSize) == int(m.buckets[i].hdib>>dibBitSize) && e.key == m.buckets[i].key {
			old := m.buckets[i].value
			m.buckets[i].value = e.value
			return old, true
		}
		if int(m.buckets[i].hdib&maxDIB) < int(e.hdib&maxDIB) {
			e, m.buckets[i] = m.buckets[i], e
		}
		i = (i + 1) & m.mask
		e.hdib = e.hdib>>dibBitSize<<dibBitSize | uint64(int(e.hdib&maxDIB)+1)&maxDIB
	}
}

// Get returns a value for a key.
// Returns false when no value has been assign for key.
func (m *shard[K, V]) Get(xxh uint64, key K) (prev V, ok bool) {
	if len(m.buckets) == 0 {
		return
	}
	hash := int(xxh >> dibBitSize)
	i := hash & m.mask
	for {
		e := m.buckets[i]
		if int(e.hdib&maxDIB) == 0 {
			return
		}
		if int(e.hdib>>dibBitSize) == hash && e.key == key {
			return e.value, true
		}
		i = (i + 1) & m.mask
	}
}

// Len returns the number of values in map.
func (m *shard[K, V]) Len() int {
	return m.length
}

// Delete deletes a value for a key.
// Returns the deleted value, or false when no value was assigned.
func (m *shard[K, V]) Delete(xxh uint64, key K) (v V, ok bool) {
	if len(m.buckets) == 0 {
		return
	}
	hash := int(xxh >> dibBitSize)
	i := hash & m.mask
	for {
		if int(m.buckets[i].hdib&maxDIB) == 0 {
			return
		}
		if int(m.buckets[i].hdib>>dibBitSize) == hash && m.buckets[i].key == key {
			old := m.buckets[i].value
			m.remove(i)
			return old, true
		}
		i = (i + 1) & m.mask
	}
}

func (m *shard[K, V]) remove(i int) {
	m.buckets[i].hdib = m.buckets[i].hdib>>dibBitSize<<dibBitSize | uint64(0)&maxDIB
	for {
		pi := i
		i = (i + 1) & m.mask
		if int(m.buckets[i].hdib&maxDIB) <= 1 {
			m.buckets[pi] = entry[K, V]{}
			break
		}
		m.buckets[pi] = m.buckets[i]
		m.buckets[pi].hdib = m.buckets[pi].hdib>>dibBitSize<<dibBitSize | uint64(int(m.buckets[pi].hdib&maxDIB)-1)&maxDIB
	}
	m.length--
	if len(m.buckets) > m.cap && m.length <= m.shrinkAt {
		m.resize(m.length)
	}
}

// Range iterates over all key/values.
// It's not safe to call or Set or Delete while ranging.
func (m *shard[K, V]) Range(iter func(key K, value V) bool) {
	for i := 0; i < len(m.buckets); i++ {
		if int(m.buckets[i].hdib&maxDIB) > 0 {
			if !iter(m.buckets[i].key, m.buckets[i].value) {
				return
			}
		}
	}
}

// GetPos gets a single keys/value nearby a position
// The pos param can be any valid uint64. Useful for grabbing a random item
// from the map.
// It's not safe to call or Set or Delete while ranging.
func (m *shard[K, V]) GetPos(pos uint64) (key K, value V, ok bool) {
	for i := 0; i < len(m.buckets); i++ {
		index := (pos + uint64(i)) & uint64(m.mask)
		if int(m.buckets[index].hdib&maxDIB) > 0 {
			return m.buckets[index].key, m.buckets[index].value, true
		}
	}
	return
}
