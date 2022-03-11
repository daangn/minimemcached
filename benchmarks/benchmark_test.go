package benchmarks

import (
	"fmt"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/daangn/minimemcached"
)

const (
	key   string = "testKey"
	value string = "testValue"
)

func BenchmarkMemcached(b *testing.B) {
	mc := memcache.New("127.0.0.1:54008")

	for i := 0; i < b.N; i++ {
		item := &memcache.Item{
			Key:   key,
			Value: []byte(value),
		}

		if err := mc.Set(item); err != nil {
			b.Errorf("err set: %v", err)
			return
		}

		res, err := mc.Get(key)
		if err != nil {
			b.Errorf("err get: %v", err)
			return
		}

		if res == nil {
			b.Errorf("get res nil, err: %v", err)
			return
		}

		if err := mc.Delete(key); err != nil {
			b.Errorf("err delete: %v", err)
			return
		}
	}
}

func BenchmarkMinimemcached(b *testing.B) {
	mm, err := minimemcached.Run(&minimemcached.Config{})
	if err != nil {
		b.Fatalf("err run: %v", err)
	}

	defer mm.Close()

	mc := memcache.New(fmt.Sprintf("127.0.0.1:%d", mm.Port()))

	for i := 0; i < b.N; i++ {
		item := &memcache.Item{
			Key:   key,
			Value: []byte(value),
		}

		if err := mc.Set(item); err != nil {
			b.Errorf("err set: %v", err)
			return
		}

		res, err := mc.Get(key)
		if err != nil {
			b.Errorf("err get: %v", err)
			return
		}

		if res == nil {
			b.Errorf("get res nil, err: %v", err)
			return
		}

		if err := mc.Delete(key); err != nil {
			b.Errorf("err delete: %v", err)
			return
		}
	}
}
