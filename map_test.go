// Copyright 2019 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an ISC-style
// license that can be found in the LICENSE file.

package shardmap

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

type keyT = string
type valueT = string

func k(key int) keyT {
	return strconv.FormatInt(int64(key), 10)
}

func add(x keyT, delta int) string {
	i, err := strconv.ParseInt(x, 10, 64)
	if err != nil {
		panic(err)
	}
	return k(int(i + int64(delta)))
}

// /////////////////////////
func random(N int, perm bool) []keyT {
	nums := make([]keyT, N)
	if perm {
		for i, x := range rand.Perm(N) {
			nums[i] = k(x)
		}
	} else {
		m := make(map[keyT]bool)
		for len(m) < N {
			m[k(int(rand.Uint64()))] = true
		}
		var i int
		for k := range m {
			nums[i] = k
			i++
		}
	}
	return nums
}

func shuffle(nums []keyT) {
	for i := range nums {
		j := rand.Intn(i + 1)
		nums[i], nums[j] = nums[j], nums[i]
	}
}

func init() {
	//var seed int64 = 1519776033517775607
	seed := (time.Now().UnixNano())
	println("seed:", seed)
	rand.Seed(seed)
}

func TestRandomData(t *testing.T) {
	N := 10000
	start := time.Now()
	for time.Since(start) < time.Second*2 {
		nums := random(N, true)
		var m *Map[string, string]
		switch rand.Int() % 5 {
		default:
			m = New[string, string](N / ((rand.Int() % 3) + 1))
		case 1, 2:
			m = New[string, string](0)
		}
		v, ok := m.Get(k(999))
		if ok || v != "" {
			t.Fatalf("expected %v, got %v", 0, v)
		}
		v, ok = m.Delete(k(999))
		if ok || v != "" {
			t.Fatalf("expected %v, got %v", 0, v)
		}
		if m.Len() != 0 {
			t.Fatalf("expected %v, got %v", 0, m.Len())
		}
		// set a bunch of items
		for i := 0; i < len(nums); i++ {
			v, ok := m.Set(nums[i], nums[i])
			if ok || v != "" {
				t.Fatalf("expected %v, got %v", 0, v)
			}
		}
		if m.Len() != N {
			t.Fatalf("expected %v, got %v", N, m.Len())
		}
		// retrieve all the items
		shuffle(nums)
		for i := 0; i < len(nums); i++ {
			v, ok := m.Get(nums[i])
			if !ok || v == "" || v != nums[i] {
				t.Fatalf("expected %v, got %v", nums[i], v)
			}
		}
		// replace all the items
		shuffle(nums)
		for i := 0; i < len(nums); i++ {
			v, ok := m.Set(nums[i], add(nums[i], 1))
			if !ok || v != nums[i] {
				t.Fatalf("expected %v, got %v", nums[i], v)
			}
		}
		if m.Len() != N {
			t.Fatalf("expected %v, got %v", N, m.Len())
		}
		// retrieve all the items
		shuffle(nums)
		for i := 0; i < len(nums); i++ {
			v, ok := m.Get(nums[i])
			if !ok || v != add(nums[i], 1) {
				t.Fatalf("expected %v, got %v", add(nums[i], 1), v)
			}
		}
		// remove half the items
		shuffle(nums)
		for i := 0; i < len(nums)/2; i++ {
			v, ok := m.Delete(nums[i])
			if !ok || v != add(nums[i], 1) {
				t.Fatalf("expected %v, got %v", add(nums[i], 1), v)
			}
		}
		if m.Len() != N/2 {
			t.Fatalf("expected %v, got %v", N/2, m.Len())
		}
		// check to make sure that the items have been removed
		for i := 0; i < len(nums)/2; i++ {
			v, ok := m.Get(nums[i])
			if ok || v != "" {
				t.Fatalf("expected %v, got %v", "", v)
			}
		}
		// check the second half of the items
		for i := len(nums) / 2; i < len(nums); i++ {
			v, ok := m.Get(nums[i])
			if !ok || v != add(nums[i], 1) {
				t.Fatalf("expected %v, got %v", add(nums[i], 1), v)
			}
		}
		// try to delete again, make sure they don't exist
		for i := 0; i < len(nums)/2; i++ {
			v, ok := m.Delete(nums[i])
			if ok || v != "" {
				t.Fatalf("expected %v, got %v", nil, v)
			}
		}
		if m.Len() != N/2 {
			t.Fatalf("expected %v, got %v", N/2, m.Len())
		}
		m.Range(func(key keyT, value valueT) bool {
			if value != add(key, 1) {
				t.Fatalf("expected %v, got %v", add(key, 1), value)
			}
			return true
		})
		var n int
		m.Range(func(key keyT, value valueT) bool {
			n++
			return false
		})
		if n != 1 {
			t.Fatalf("expected %v, got %v", 1, n)
		}
		for i := len(nums) / 2; i < len(nums); i++ {
			v, ok := m.Delete(nums[i])
			if !ok || v != add(nums[i], 1) {
				t.Fatalf("expected %v, got %v", add(nums[i], 1), v)
			}
		}
	}
}

func TestMutate(t *testing.T) {
	m := New[string, string](0)
	m.Set("hello", "world")
	var prev string
	delta := m.Mutate("hello", func(value string, ok bool) (string, bool) {
		prev = value
		return "planet", true
	})
	if delta != 0 {
		t.Fatal("expected 0 detla")
	}
	if prev != "world" {
		t.Fatalf("expected '%v', got '%v'", "world", prev)
	}
	if v, _ := m.Get("hello"); v != "planet" {
		t.Fatalf("expected '%v', got '%v'", "planet", v)
	}

	delta = m.Mutate("hello", func(value string, ok bool) (string, bool) {
		prev = value
		return "world", true
	})
	if delta != 0 {
		t.Fatal("expected 0 delta")
	}
	if prev != "planet" {
		t.Fatalf("expected '%v', got '%v'", "planet", prev)
	}

	delta = m.Mutate("hello", func(value string, ok bool) (string, bool) {
		prev = ""
		return value, true
	})
	if delta != 0 {
		t.Fatal("expected 0 delta")
	}
	if prev != "" {
		t.Fatalf("expected '%v', got '%v'", "", prev)
	}
	if v, _ := m.Get("hello"); v != "world" {
		t.Fatalf("expected '%v', got '%v'", "world", v)
	}

	delta = m.Mutate("hi", func(value string, ok bool) (string, bool) {
		prev = value
		return "world", true
	})
	if delta != 1 {
		t.Fatal("expected 1 delta")
	}
	if prev != "" {
		t.Fatalf("expected '%v', got '%v'", "", prev)
	}

	m.Set("hello", "world")
	delta = m.Mutate("hello", func(value string, ok bool) (string, bool) {
		prev = value
		return "", false
	})
	if delta != -1 {
		t.Fatal("expected -1 delta")
	}
	if prev != "world" {
		t.Fatalf("expected '%v', got '%v'", "world", prev)
	}

	m.Set("hello", "world")
	delta = m.Mutate("hello", func(value string, ok bool) (string, bool) {
		if !ok {
			t.Fatal("expected true")
		}
		if value != "world" {
			t.Fatalf("expected '%v', got '%v'", "world", value)
		}
		prev = ""
		return value, true
	})
	if delta == -1 {
		t.Fatal("expected 0 delta")
	}
	if prev != "" {
		t.Fatalf("expected '%v', got '%v'", "", prev)
	}
	prev, ok := m.Get("hello")
	if !ok {
		t.Fatal("expected true")
	}
	if prev != "world" {
		t.Fatalf("expected '%v', got '%v'", "world", prev)
	}

}

func TestClear(t *testing.T) {
	m := New[string, int](0)
	for i := 0; i < 1000; i++ {
		m.Set(fmt.Sprintf("%d", i), i)
	}
	if m.Len() != 1000 {
		t.Fatalf("expected '%v', got '%v'", 1000, m.Len())
	}
	m.Clear()
	if m.Len() != 0 {
		t.Fatalf("expected '%v', got '%v'", 0, m.Len())
	}

}

// see https://github.com/cornelk/hashmap/issues/73
func BenchmarkHashMap_RaceCase1(b *testing.B) {
	const (
		elementNum0 = 1024
		iter0       = 8
	)
	b.StopTimer()
	wg := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		m := New[int, int](0)
		b.StartTimer()
		for k := 0; k < iter0; k++ {
			wg.Add(1)
			go func(l, h int) {
				for j := l; j < h; j++ {
					m.Set(j, j)
				}
				for j := l; j < h; j++ {
					_, a := m.Get(j)
					if !a {
						b.Error("key doesn't exist", j)
					}
				}
				for j := l; j < h; j++ {
					x, _ := m.Get(j)
					if x != j {
						b.Error("incorrect value", j, x)
					}
				}
				wg.Done()
			}(k*elementNum0, (k+1)*elementNum0)
		}
		wg.Wait()
		b.StopTimer()
	}
}

func BenchmarkHashMap_RaceCase3(b *testing.B) {
	const (
		elementNum0 = 1024
		iter0       = 8
	)
	b.StopTimer()
	wg := &sync.WaitGroup{}
	for a := 0; a < b.N; a++ {
		m := New[int, int](0)
		b.StartTimer()
		for j := 0; j < iter0; j++ {
			wg.Add(1)
			go func(l, h int) {
				defer wg.Done()
				for i := l; i < h; i++ {
					m.Set(i, i)
				}

				for i := l; i < h; i++ {
					_, x := m.Get(i)
					if !x {
						b.Errorf("not put: %v\n", i)
					}
				}
				for i := l; i < h; i++ {
					m.Delete(i)

				}
				for i := l; i < h; i++ {
					_, x := m.Get(i)
					if x {
						b.Errorf("not removed: %v\n", i)
					}
				}

			}(j*elementNum0, (j+1)*elementNum0)
		}
		wg.Wait()
		b.StopTimer()
	}

}
