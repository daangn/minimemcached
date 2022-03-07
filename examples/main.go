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
