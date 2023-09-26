# `shardmap`

[![GoDoc](https://img.shields.io/badge/api-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/phuslu/shardmap)

A simple and efficient thread-safe sharded hashmap for Go.
This is an alternative to the standard Go map and `sync.Map`, and is optimized
for when your map needs to perform lots of concurrent reads and writes.

Under the hood `shardmap` uses 
[robinhood hashmap](https://en.wikipedia.org/wiki/Hash_table#Robin_Hood_hashing) and 
[wyhash](https://github.com/zeebo/wyhash).

# Getting Started

## Installing

To start using `shardmap`, install Go and run `go get`:

```sh
$ go get -u github.com/phuslu/shardmap
```

This will retrieve the library.

## Usage

The `Map` type works similar to a standard Go map, and includes four methods:
`Set`, `Get`, `Delete`, `Len`.

```go
m := shardmap.New[string, string](0)
m.Set("Hello", "Dolly!")
val, _ := m.Get("Hello")
fmt.Printf("%v\n", val)
val, _ = m.Delete("Hello")
fmt.Printf("%v\n", val)
val, _ = m.Get("Hello")
fmt.Printf("%v\n", val)

// Output:
// Dolly!
// Dolly!
// 
```

## Performance

Benchmarking concurrent SET, GET, RANGE, and DELETE operations for 
    `sync.Map`, `map[string]interface{}`, `github.com/phuslu/shardmap`. 

```
go version go1.21.1 linux/amd64 (Intel(R) Xeon(R) Silver 4216 CPU @ 2.10GHz)

     number of cpus: 64
     number of keys: 10000000
            keysize: 16
        random seed: 1695723704044340631

-- github.com/orcaman/concurrent-map/v2 --
set: 10,000,000 ops over 64 threads in 1572ms, 6,363,096/sec, 157 ns/op
get: 10,000,000 ops over 64 threads in 274ms, 36,550,366/sec, 27 ns/op
del: 10,000,000 ops over 64 threads in 827ms, 12,094,724/sec, 82 ns/op
rng:       100 ops over 64 threads in 0ms, 425,928/sec, 2347 ns/op

-- tailscale.com/syncs --
set: 10,000,000 ops over 64 threads in 428ms, 23,350,888/sec, 42 ns/op
get: 10,000,000 ops over 64 threads in 103ms, 97,508,463/sec, 10 ns/op
del: 10,000,000 ops over 64 threads in 141ms, 70,870,381/sec, 14 ns/op

-- github.com/alphadose/haxmap --
set: 10,000,000 ops over 64 threads in 12678ms, 788,770/sec, 1267 ns/op
get: 10,000,000 ops over 64 threads in 80ms, 125,642,501/sec, 7 ns/op
del: 10,000,000 ops over 64 threads in 1582ms, 6,320,754/sec, 158 ns/op
rng:       100 ops over 64 threads in 0ms, 486,807/sec, 2054 ns/op

-- github.com/phuslu/shardmap --
set: 10,000,000 ops over 64 threads in 425ms, 23,521,366/sec, 42 ns/op
get: 10,000,000 ops over 64 threads in 79ms, 125,923,770/sec, 7 ns/op
del: 10,000,000 ops over 64 threads in 120ms, 83,307,985/sec, 12 ns/op
rng:       100 ops over 64 threads in 1ms, 77,454/sec, 12910 ns/op
```

## License

`shardmap` source code is available under the MIT [License](/LICENSE).
