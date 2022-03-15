package minimemcached

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/bradfitz/gomemcache/memcache"
)

var cfg = &Config{
	Port: 0,
}

func validateGetItemResult(want *memcache.Item, got *memcache.Item) error {
	if !bytes.Equal(want.Value, got.Value) {
		return fmt.Errorf("value: got %v, want %v", got.Value, want.Value)
	}
	if want.Flags != got.Flags {
		return fmt.Errorf("flags: got %d, want %d", got.Flags, want.Flags)
	}
	return nil
}

func TestGetSet(t *testing.T) {
	key, value := "testKey", "testValue"
	expiration := 60
	item := &memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Expiration: int32(expiration),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))
	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}
	if err = validateGetItemResult(item, res); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestGetMulti(t *testing.T) {
	key1, value1 := "testKey1", "testValue1"
	key2, value2 := "testKey2", "testValue2"
	item1 := &memcache.Item{
		Key:   key1,
		Value: []byte(value1),
	}
	item2 := &memcache.Item{
		Key:   key2,
		Value: []byte(value2),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))
	if err := mc.Set(item1); err != nil {
		t.Errorf("err: %v", err)
		return
	}
	if err := mc.Set(item2); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.GetMulti([]string{key1, key2})
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}
	res1 := res[key1]
	if res1 == nil {
		t.Errorf("res1 nil")
		return
	}
	if err = validateGetItemResult(item1, res1); err != nil {
		t.Errorf("%v", err)
		return
	}
	res2 := res[key2]
	if res2 == nil {
		t.Errorf("res2 nil")
		return
	}
	if err = validateGetItemResult(item2, res2); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestGetNoResult(t *testing.T) {
	key, value := "testKey", "testValue"
	wrongKey := "wrongKey"
	expiration := 60
	item := &memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Expiration: int32(expiration),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))
	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.GetMulti([]string{key, wrongKey})
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}
	it := res[key]
	if it == nil {
		t.Errorf("it nil")
		return
	}
	if err = validateGetItemResult(item, it); err != nil {
		t.Errorf("%v", err)
		return
	}
	if wrongRes := res[wrongKey]; err != nil {
		t.Errorf("wrongRes not nil. want: nil, got: %v", wrongRes)
		return
	}
}

func TestTTLInvalidation(t *testing.T) {
	key, value := "testKey", "testValue"
	expiration := 2
	item := &memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Expiration: int32(expiration),
	}

	clk := clock.NewMock()

	m, err := Run(cfg, WithClock(clk))
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))
	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	clk.Add(3 * time.Second)

	res, err := mc.Get(key)
	if err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}

	if res != nil {
		t.Errorf("res not nil, invalidation failed.")
		return
	}
}

func TestCASToken(t *testing.T) {
	key1, value1 := "testKey1", "testValue1"
	key2, value2 := "testKey2", "testValue2"
	key3, value3 := "testKey3", "testValue3"
	item1 := &memcache.Item{
		Key:   key1,
		Value: []byte(value1),
	}
	item2 := &memcache.Item{
		Key:   key2,
		Value: []byte(value2),
	}
	item3 := &memcache.Item{
		Key:   key3,
		Value: []byte(value3),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))
	if err := mc.Set(item1); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if err := mc.Set(item2); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	i1 := m.items[key1]
	if i1 == nil {
		t.Error("i1 nil")
		return
	}
	if i1.CASToken != 1 {
		t.Errorf("i1.CASToken wrong. want: 1, got: %d", i1.CASToken)
		return
	}

	i2 := m.items[key2]
	if i2 == nil {
		t.Error("i2 nil")
		return
	}
	if i2.CASToken != 2 {
		t.Errorf("i2.CASToken wrong. want: 2, got: %d", i2.CASToken)
		return
	}

	if err := mc.Set(item1); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	i1 = m.items[key1]
	if i1 == nil {
		t.Error("i1 nil")
		return
	}
	if i1.CASToken != 3 {
		t.Errorf("i1.CASToken wrong. want: 3, got: %d", i1.CASToken)
		return
	}

	if err = mc.Set(item3); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	i3 := m.items[key3]
	if i3 == nil {
		t.Error("i3 nil")
		return
	}
	if i3.CASToken != 4 {
		t.Errorf("i3.CASToken wrong. want: 4, got: %d", i3.CASToken)
		return
	}

	i2 = m.items[key2]
	if i2 == nil {
		t.Error("i2 nil")
		return
	}
	if i2.CASToken != 2 {
		t.Errorf("i2.CASToken wrong. want: 2, got: %d", i2.CASToken)
		return
	}
}

func TestAddSuccess(t *testing.T) {
	key, value := "testKey", "testValue"
	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Add(item); err != nil {
		t.Errorf("add failed. err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if res == nil {
		t.Errorf("failed to get added item. err: %v", err)
		return
	}

	if err = validateGetItemResult(item, res); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestAddFail(t *testing.T) {
	key, value := "testKey", "testValue"
	newValue := "newValue"
	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	addItem := &memcache.Item{
		Key:   key,
		Value: []byte(newValue),
	}
	if err := mc.Add(addItem); err != nil && !errors.Is(err, memcache.ErrNotStored) {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if err = validateGetItemResult(item, res); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestReplaceSuccess(t *testing.T) {
	key, value := "testKey", "testValue"
	newValue := "newValue"
	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	newItem := &memcache.Item{
		Key:   key,
		Value: []byte(newValue),
	}

	if err := mc.Replace(newItem); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if res == nil {
		t.Errorf("failed to get added item. err: %v", err)
		return
	}

	if err = validateGetItemResult(newItem, res); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestReplaceFail(t *testing.T) {
	key, value := "testKey", "testValue"
	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Replace(item); err != nil && !errors.Is(err, memcache.ErrNotStored) {
		t.Errorf("err: %v", err)
		return
	}
}

func TestReplaceFailItemInvalidated(t *testing.T) {
	key, value := "testKey", "testValue"
	item := &memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Expiration: 2,
	}

	clk := clock.NewMock()

	m, err := Run(cfg, WithClock(clk))
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	clk.Add(3 * time.Second)

	if err := mc.Replace(item); err != nil && !errors.Is(err, memcache.ErrNotStored) {
		t.Errorf("err: %v", err)
		return
	} else if err == nil {
		t.Errorf("item must be invalidated")
		return
	}
}

func writeAppend(port uint16, key string, value []byte) error {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	message := fmt.Sprintf("%s %s %d %d %d", appendCmd, key, 0, 0, len(value))

	if _, err := conn.Write(append([]byte(message), crlf...)); err != nil {
		return err
	}

	if _, err := conn.Write(append(value, crlf...)); err != nil {
		return err
	}

	rw := bufio.NewReader(conn)
	resp, _ := rw.ReadSlice('\n')
	switch {
	case bytes.Equal(resp, resultStored):
		return nil
	case bytes.Equal(resp, resultNotStored):
		return memcache.ErrCacheMiss
	}
	return nil
}

func writePrepend(port uint16, key string, value []byte) error {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	message := fmt.Sprintf("%s %s %d %d %d", prependCmd, key, 0, 0, len(value))

	if _, err := conn.Write(append([]byte(message), crlf...)); err != nil {
		return err
	}

	if _, err := conn.Write(append(value, crlf...)); err != nil {
		return err
	}

	rw := bufio.NewReader(conn)
	resp, _ := rw.ReadSlice('\n')
	switch {
	case bytes.Equal(resp, resultStored):
		return nil
	case bytes.Equal(resp, resultNotStored):
		return memcache.ErrCacheMiss
	}
	return nil
}

func TestAppendSuccess(t *testing.T) {
	key, value := "testKey", "testValue"
	appendValue, resultValue := "Append", "testValueAppend"
	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if err := writeAppend(m.Port(), key, []byte(appendValue)); err != nil {
		t.Errorf("failed to append. err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if res == nil {
		t.Errorf("failed to get item. err: %v", err)
		return
	}

	if !bytes.Equal([]byte(resultValue), res.Value) {
		t.Errorf("append failed. want: %s, got: %s", resultValue, string(res.Value))
		return
	}
}

func TestAppendFail(t *testing.T) {
	key, value := "testKey", "testValue"

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := writeAppend(m.Port(), key, []byte(value)); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("failed to append. err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}

	if res != nil {
		t.Errorf("invalid append operation.")
		return
	}
}

func TestPrependSuccess(t *testing.T) {
	key, value := "testKey", "TestValue"
	prependValue, resultValue := "Prepend", "PrependTestValue"
	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if err := writePrepend(m.Port(), key, []byte(prependValue)); err != nil {
		t.Errorf("failed to prepend. err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if res == nil {
		t.Errorf("failed to get item. err: %v", err)
		return
	}

	if !bytes.Equal([]byte(resultValue), res.Value) {
		t.Errorf("prepend failed. want: %s, got: %s", resultValue, string(res.Value))
		return
	}
}

func TestPrependFail(t *testing.T) {
	key, value := "testKey", "testValue"

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := writePrepend(m.Port(), key, []byte(value)); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("failed to prepend. err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}

	if res != nil {
		t.Errorf("invalid prepend operation.")
		return
	}
}

func TestDeleteSuccess(t *testing.T) {
	key, value := "testKey", "testValue"
	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if err := mc.Delete(key); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}

	if res != nil {
		t.Errorf("invalid delete operation.")
		return
	}
}

func TestDeleteFail(t *testing.T) {
	key := "testKey"

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Delete(key); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}
}

func TestIncrSuccess(t *testing.T) {
	key, value := "testKey", "10"
	const incrValue uint64 = 10
	const incrementedValue = 20

	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	newValue, err := mc.Increment(key, incrValue)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if newValue != incrementedValue {
		t.Errorf("wrong newValue. want: %d, got: %d", incrementedValue, newValue)
		return
	}

	updatedItem, err := mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	newValue, err = strconv.ParseUint(string(updatedItem.Value), 10, 64)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if newValue != incrementedValue {
		t.Errorf("wrong newValue. want: %d, got: %d", incrementedValue, newValue)
		return
	}
}

func TestIncrFailNonNumericValue(t *testing.T) {
	key, value := "testKey", "nonNumericValue"
	const incrValue uint64 = 10

	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if _, err := mc.Increment(key, incrValue); err != nil && err.Error() != "memcache: client error: cannot increment or decrement non-numeric value" {
		t.Errorf("err: %v", err)
		return
	}
}

func TestIncrFailNotFound(t *testing.T) {
	key := "testKey"
	const incrValue uint64 = 10

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if _, err := mc.Increment(key, incrValue); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}
}

func TestIncrMaxValueOverflowBecomesZero(t *testing.T) {
	key, value := "testKey", "1"
	const incrValue = math.MaxUint64
	const incrementedValue = 0

	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	newValue, err := mc.Increment(key, incrValue)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if newValue != incrementedValue {
		t.Errorf("invalid incr operation. want: %d, got: %d", incrementedValue, newValue)
	}

}

func TestDecrSuccess(t *testing.T) {
	key, value := "testKey", "30"
	const decrValue uint64 = 10
	const decrementedValue = 20

	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	newValue, err := mc.Decrement(key, decrValue)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if newValue != decrementedValue {
		t.Errorf("wrong newValue. want: %d, got: %d", decrementedValue, newValue)
		return
	}

	updatedItem, err := mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	newValue, err = strconv.ParseUint(string(updatedItem.Value), 10, 64)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if newValue != decrementedValue {
		t.Errorf("wrong newValue. want: %d, got: %d", decrementedValue, newValue)
		return
	}
}

func TestDecrFailNonNumericValue(t *testing.T) {
	key, value := "testKey", "nonNumericValue"
	const decrValue uint64 = 10

	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if _, err := mc.Decrement(key, decrValue); err != nil && err.Error() != "memcache: client error: cannot increment or decrement non-numeric value" {
		t.Errorf("err: %v", err)
		return
	}
}

func TestDecrFailNotFound(t *testing.T) {
	key := "testKey"
	const decrValue uint64 = 10

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if _, err := mc.Decrement(key, decrValue); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}
}

func TestDecrLowestValueIsZero(t *testing.T) {
	key, value := "testKey", "30"
	const decrValue uint64 = 100

	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	newValue, err := mc.Decrement(key, decrValue)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if newValue != 0 {
		t.Errorf("invalid decr operation. want: 0, got: %d", newValue)
	}
}

func TestTouchSuccess(t *testing.T) {
	key, value := "testKey", "testValue"
	expiration := 60
	item := &memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Expiration: int32(expiration),
	}

	clk := clock.NewMock()

	m, err := Run(cfg, WithClock(clk))
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	const expTime int32 = 2
	if err := mc.Touch(key, expTime); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	clk.Add(3 * time.Second)

	res, err := mc.Get(key)
	if err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}

	if res != nil {
		t.Errorf("res not nil, touch command failed.")
		return
	}
}

func TestTouchFailNotFound(t *testing.T) {
	key := "testKey"
	const expTime int32 = 2

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Touch(key, expTime); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}
}

func TestFlushAll(t *testing.T) {
	key, value := "testKey", "testValue"
	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if err := mc.FlushAll(); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	}

	if res != nil {
		t.Errorf("res not nil, invalidation failed.")
		return
	}
}

func TestCASSuccess(t *testing.T) {
	key, value := "testKey", "testValue"

	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	savedItem, err := mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	newValue := "newValue"

	savedItem.Value = []byte(newValue)
	if err := mc.CompareAndSwap(savedItem); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if err = validateGetItemResult(savedItem, res); err != nil {
		t.Errorf("%v", err)
		return
	}

}

func TestCASFailInvalidCASToken(t *testing.T) {
	key, value := "testKey", "testValue"
	newValue := "newValue"

	item := &memcache.Item{
		Key:   key,
		Value: []byte(value),
	}

	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	savedItem, err := mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	replacedValue := "replacedValue"
	replaceItem := &memcache.Item{
		Key:   key,
		Value: []byte(replacedValue),
	}

	if err := mc.Replace(replaceItem); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	savedItem.Value = []byte(newValue)

	if err := mc.CompareAndSwap(savedItem); err != nil && !errors.Is(err, memcache.ErrCASConflict) {
		t.Errorf("err: %v", err)
		return
	}

	res, err := mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	if err = validateGetItemResult(replaceItem, res); err != nil {
		t.Errorf("%v", err)
		return
	}
}

func TestCASFailNotFound(t *testing.T) {
	key, value := "testKey", "testValue"
	const expTime int32 = 2

	item := &memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Expiration: expTime,
	}

	clk := clock.NewMock()

	m, err := Run(cfg, WithClock(clk))
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Set(item); err != nil {
		t.Errorf("err: %v", err)
		return
	}

	item, err = mc.Get(key)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	clk.Add(3 * time.Second)

	newValue := "newValue"
	item.Value = []byte(newValue)

	if err := mc.CompareAndSwap(item); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		t.Errorf("err: %v", err)
		return
	} else if err == nil {
		t.Errorf("item must be invalidated")
		return
	}
}

func TestVersion(t *testing.T) {
	m, err := Run(cfg)
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}

	defer m.Close()

	mc := memcache.New(fmt.Sprintf(":%d", m.Port()))

	if err := mc.Ping(); err != nil {
		t.Errorf("err: %v", err)
		return
	}
}
