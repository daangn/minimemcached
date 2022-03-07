# minimemcached

Minimemcached is a Memcached server for written in Go, aimed for unittests in Go projects.

---

When you have to test codes that use Memcached server, running actual Memcached server instance could be quite expensive,
depending on your environment.

Minimemcached aims to solve this problem by implementing Memcached's TCP interface 100% in Go,
and works perfectly well with [gomemcache](https://github.com/bradfitz/gomemcache), a memcache client for Go.

<details><summary>Implemented commands</summary>

<p>

- get
- gets
- cas
- set
- touch
- add
- replace
- append
- prepend
- delete
- incr
- decr
- flush_all
- version

</p>
</details>

## Setup

- To use Minimemcached, you can import it as below.
  You can also view [example code](https://github.com/daangn/minimemcached/blob/main/examples/main.go).

```go
package main

import (
 "fmt"

 "github.com/bradfitz/gomemcache/memcache"

 "github.com/daangn/minimemcached"
)

func main() {
 cfg := &minimemcached.Config{
   Port: 8080,
 }
 m, err := minimemcached.Run(cfg)
 if err != nil {
   fmt.Println("failed to start mini-memcached server.")
   return
 }

 defer m.Close()

 fmt.Println("mini-memcached started")

 mc := memcache.New(fmt.Sprintf("localhost:%d", m.Port()))
 err = mc.Set(&memcache.Item{Key: "foo", Value: []byte("my value"), Expiration: int32(60)})
 if err != nil {
   fmt.Printf("err(set): %v\n", err)
 }

 it, err := mc.Get("foo")
 if err != nil {
   fmt.Printf("err(get): %v\n", err)
 } else {
   fmt.Printf("key: %s, value: %s\n", it.Key, it.Value)
   fmt.Printf("value: %s\n", string(it.Value))
 }
}
```

---

If you want to contribute to Minimemcached, feel free to create an issue or pull request!
