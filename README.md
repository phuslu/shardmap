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

Benchmarking concurrent SET, GET, and DELETE operations for 
    `sync.Map`, `orcaman/concurrent-map`, `phuslu/shardmap`. 

```
go version go1.21.1 linux/amd64 (Intel(R) Xeon(R) Silver 4216 CPU @ 2.10GHz)

     number of cpus: 64
     number of keys: 10000000
            keysize: 16
        random seed: 1695872462865986967

-- sync.Map --
set: 10,000,000 ops over 64 threads in 16141ms, 619,553/sec, 1614 ns/op
get: 10,000,000 ops over 64 threads in 6919ms, 1,445,229/sec, 691 ns/op
del: 10,000,000 ops over 64 threads in 5016ms, 1,993,464/sec, 501 ns/op

-- github.com/orcaman/concurrent-map/v2 --
set: 10,000,000 ops over 64 threads in 1327ms, 7,533,463/sec, 132 ns/op
get: 10,000,000 ops over 64 threads in 252ms, 39,742,423/sec, 25 ns/op
del: 10,000,000 ops over 64 threads in 769ms, 12,995,480/sec, 76 ns/op

-- tailscale.com/syncs --
set: 10,000,000 ops over 64 threads in 473ms, 21,120,763/sec, 47 ns/op
get: 10,000,000 ops over 64 threads in 107ms, 93,572,966/sec, 10 ns/op
del: 10,000,000 ops over 64 threads in 124ms, 80,512,652/sec, 12 ns/op

-- github.com/alphadose/haxmap --
set: 10,000,000 ops over 64 threads in 13285ms, 752,728/sec, 1328 ns/op
get: 10,000,000 ops over 64 threads in 84ms, 119,259,602/sec, 8 ns/op
del: 10,000,000 ops over 64 threads in 1641ms, 6,092,923/sec, 164 ns/op

-- github.com/phuslu/shardmap --
set: 10,000,000 ops over 64 threads in 281ms, 35,588,233/sec, 28 ns/op
get: 10,000,000 ops over 64 threads in 80ms, 125,519,306/sec, 7 ns/op
del: 10,000,000 ops over 64 threads in 121ms, 82,783,512/sec, 12 ns/op
```

## License

`shardmap` source code is available under the MIT [License](/LICENSE).
