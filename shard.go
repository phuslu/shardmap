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
	key   K // user key
	value V // user value
}

// Map is a hashmap. Like map[comparable]any
type shard[K comparable, V any] struct {
	hdib     []uint64 // bitfield { hash:48 dib:16 }
	buckets  []entry[K, V]
	cap      int
	length   int
	mask     int
	growAt   int
	shrinkAt int
}

func (m *shard[K, V]) init(cap int) {
	m.cap = cap
	m.length = 0
	sz := 8
	for sz < m.cap {
		sz *= 2
	}
	if m.cap > 0 {
		m.cap = sz
	}
	m.hdib = make([]uint64, sz)
	m.buckets = make([]entry[K, V], sz)
	m.mask = len(m.buckets) - 1
	m.growAt = int(float64(len(m.buckets)) * loadFactor)
	m.shrinkAt = int(float64(len(m.buckets)) * (1 - loadFactor))
}

func (m *shard[K, V]) resize(newCap int) {
	var nmap shard[K, V]
	nmap.init(newCap)
	for i := 0; i < len(m.buckets); i++ {
		if int(m.hdib[i]&maxDIB) > 0 {
			nmap.set(int(m.hdib[i]>>dibBitSize), m.buckets[i].key, m.buckets[i].value)
		}
	}
	cap := m.cap
	*m = nmap
	m.cap = cap
}

// Set assigns a value to a key.
// Returns the previous value, or false when no value was assigned.
func (m *shard[K, V]) Set(xxh uint64, key K, value V) (V, bool) {
	if len(m.buckets) == 0 {
		m.init(0)
	}
	if m.length >= m.growAt {
		m.resize(len(m.buckets) * 2)
	}
	return m.set(int(xxh>>dibBitSize), key, value)
}

func (m *shard[K, V]) set(hash int, key K, value V) (prev V, ok bool) {
	hdib := uint64(hash)<<dibBitSize | uint64(1)&maxDIB
	e := entry[K, V]{key, value}
	i := int(hdib>>dibBitSize) & m.mask
	for {
		if int(m.hdib[i]&maxDIB) == 0 {
			m.hdib[i] = hdib
			m.buckets[i] = e
			m.length++
			return
		}
		if int(hdib>>dibBitSize) == int(m.hdib[i]>>dibBitSize) && e.key == m.buckets[i].key {
			old := m.buckets[i].value
			m.hdib[i] = hdib
			m.buckets[i].value = e.value
			return old, true
		}
		if int(m.hdib[i]&maxDIB) < int(hdib&maxDIB) {
			hdib, m.hdib[i] = m.hdib[i], hdib
			e, m.buckets[i] = m.buckets[i], e
		}
		i = (i + 1) & m.mask
		hdib = hdib>>dibBitSize<<dibBitSize | uint64(int(hdib&maxDIB)+1)&maxDIB
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
		if int(m.hdib[i]&maxDIB) == 0 {
			return
		}
		if int(m.hdib[i]>>dibBitSize) == hash && m.buckets[i].key == key {
			return m.buckets[i].value, true
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
		if int(m.hdib[i]&maxDIB) == 0 {
			return
		}
		if int(m.hdib[i]>>dibBitSize) == hash && m.buckets[i].key == key {
			old := m.buckets[i].value
			m.remove(i)
			return old, true
		}
		i = (i + 1) & m.mask
	}
}

func (m *shard[K, V]) remove(i int) {
	m.hdib[i] = m.hdib[i]>>dibBitSize<<dibBitSize | uint64(0)&maxDIB
	for {
		pi := i
		i = (i + 1) & m.mask
		if int(m.hdib[i]&maxDIB) <= 1 {
			m.buckets[pi] = entry[K, V]{}
			m.hdib[pi] = 0
			break
		}
		m.buckets[pi] = m.buckets[i]
		m.hdib[pi] = m.hdib[i]>>dibBitSize<<dibBitSize | uint64(int(m.hdib[i]&maxDIB)-1)&maxDIB
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
		if int(m.hdib[i]&maxDIB) > 0 {
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
		if int(m.hdib[index]&maxDIB) > 0 {
			return m.buckets[index].key, m.buckets[index].value, true
		}
	}
	return
}
