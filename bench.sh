#!/bin/sh

rm -rf bench
mkdir -p bench
cd bench

cat << EOF > main.go

package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/alphadose/haxmap"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/phuslu/shardmap"
	"github.com/tidwall/lotsa"
	"github.com/zeebo/xxh3"
	tailscalesyncs "tailscale.com/syncs"
)

func randKey(rnd *rand.Rand, n int) string {
	s := make([]byte, n)
	rnd.Read(s)
	for i := 0; i < n; i++ {
		s[i] = 'a' + (s[i] % 26)
	}
	return string(s)
}

func main() {

	seed := time.Now().UnixNano()
	// println(seed)
	rng := rand.New(rand.NewSource(seed))
	N := 10_000_000
	K := 16

	fmt.Printf("\n")
	fmt.Printf("go version %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("\n")
	fmt.Printf("     number of cpus: %d\n", runtime.NumCPU())
	fmt.Printf("     number of keys: %d\n", N)
	fmt.Printf("            keysize: %d\n", K)
	fmt.Printf("        random seed: %d\n", seed)

	fmt.Printf("\n")

	keysm := make(map[string]bool, N)
	for len(keysm) < N {
		keysm[randKey(rng, K)] = true
	}
	keys := make([]string, 0, N)
	for key := range keysm {
		keys = append(keys, key)
	}

	lotsa.Output = os.Stdout
	// lotsa.MemUsage = true

	println("-- sync.Map --")
	var sm sync.Map
	print("set: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		sm.Store(keys[i], i)
	})

	print("get: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		v, _ := sm.Load(keys[i])
		if v.(int) != i {
			panic("bad news")
		}
	})
	print("del: ")
	var mu sync.Mutex
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		mu.Lock()
		sm.Delete(keys[i])
		mu.Unlock()
	})
	println()

	println("-- github.com/orcaman/concurrent-map/v2 --")
	cmap := cmap.New[int]()
	print("set: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		cmap.Set(keys[i], i)
	})

	print("get: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		v, _ := cmap.Get(keys[i])
		if v != i {
			panic("bad news")
		}
	})
	print("del: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		cmap.Remove(keys[i])
	})

	println()

	println("-- tailscale.com/syncs --")
	var tmap = tailscalesyncs.NewShardedMap[string, int](1024, func(key string) (i int) {
		i = int(xxh3.HashString(key)) % 1024
		if i < 0 {
			i = -i
		}
		return
	})
	print("set: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		tmap.Set(keys[i], i)
	})
	print("get: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		v := tmap.Get(keys[i])
		if v != i {
			panic("bad news")
		}
	})
	print("del: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		tmap.Delete(keys[i])
	})

	println()

	println("-- github.com/alphadose/haxmap --")
	var hmap = haxmap.New[string, int]()
	print("set: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		hmap.Set(keys[i], i)
	})
	print("get: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		v, _ := hmap.Get(keys[i])
		if v != i {
			panic("bad news")
		}
	})
	print("del: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		hmap.Del(keys[i])
	})

	println()

	println("-- github.com/phuslu/shardmap --")
	com := shardmap.New[string, int](0)
	print("set: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		com.Set(keys[i], i)
	})
	print("get: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		v, _ := com.Get(keys[i])
		if v != i {
			panic("bad news")
		}
	})
	print("del: ")
	lotsa.Ops(N, runtime.NumCPU(), func(i, _ int) {
		com.Delete(keys[i])
	})

	println()

}
EOF

go mod init main
go mod tidy
go build -v
./main
