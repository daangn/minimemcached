package minimemcached

import (
	"fmt"
	"math/big"
	"net"
	"strconv"
)

// gets() handles memcached `gets` command.
func (m *MiniMemcached) gets(keys []string, conn net.Conn) {
	for _, k := range keys {
		if !isLegalKey(k) {
			_, _ = conn.Write(resultClientErrBadCliFormat)
			if m.logger.Level == Debug {
				m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
			}
			return
		}
	}
	result := make([]byte, 0)
	for _, k := range keys {
		m.invalidate(k)
		m.mu.RLock()
		item := m.items[k]
		m.mu.RUnlock()
		if item != nil {
			result = append(result, []byte(fmt.Sprintf("%s %s %d %d %d\r\n", value, k, item.flags, len(item.value), item.casToken))...)
			result = append(result, item.value...)
			result = append(result, crlf...)
		}
	}

	result = append(result, resultEnd...)
	_, _ = conn.Write(result)

	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(result)))
	}
}

// set() handles memcached `set` command.
func (m *MiniMemcached) set(key string, item *item, bytes int, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}
	if !isLegalValue(bytes, item.value) {
		_, _ = conn.Write(resultClientErrBadDataChunk)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadDataChunk)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	item.casToken = m.incrementCASToken()
	m.items[key] = item
	m.mu.Unlock()

	_, _ = conn.Write(resultStored)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultStored)))
	}
}

// add() handles memcached `add` command.
func (m *MiniMemcached) add(key string, item *item, bytes int, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}
	if !isLegalValue(bytes, item.value) {
		_, _ = conn.Write(resultClientErrBadDataChunk)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadDataChunk)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	if prevItem := m.items[key]; prevItem != nil {
		_, _ = conn.Write(resultNotStored)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotStored)))
		}
		return
	}

	item.casToken = m.incrementCASToken()
	m.items[key] = item

	_, _ = conn.Write(resultStored)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultStored)))
	}
}

// replace() handles memcached `replace` command.
func (m *MiniMemcached) replace(key string, item *item, bytes int, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}
	if !isLegalValue(bytes, item.value) {
		_, _ = conn.Write(resultClientErrBadDataChunk)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadDataChunk)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	if prevItem := m.items[key]; prevItem == nil {
		_, _ = conn.Write(resultNotStored)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotStored)))
		}
		return
	}

	item.casToken = m.incrementCASToken()
	m.items[key] = item

	_, _ = conn.Write(resultStored)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultStored)))
	}
}

// append() handles memcached `append` command.
func (m *MiniMemcached) append(key string, bytes int, value []byte, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultErr)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultErr)))
		}
		return
	}
	if !isLegalValue(bytes, value) {
		_, _ = conn.Write(resultClientErrBadDataChunk)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadDataChunk)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	prevItem := m.items[key]
	if prevItem == nil {
		_, _ = conn.Write(resultNotStored)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotStored)))
		}
		return
	}

	prevItem.casToken = m.incrementCASToken()
	prevItem.value = append(prevItem.value, value...)

	_, _ = conn.Write(resultStored)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultStored)))
	}
}

// prepend() handles memcached `prepend` command.
func (m *MiniMemcached) prepend(key string, bytes int, value []byte, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}
	if !isLegalValue(bytes, value) {
		_, _ = conn.Write(resultClientErrBadDataChunk)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadDataChunk)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	prevItem := m.items[key]
	if prevItem == nil {
		_, _ = conn.Write(resultNotStored)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotStored)))
		}
		return
	}

	prevItem.casToken = m.incrementCASToken()
	prevItem.value = append(value, prevItem.value...)

	_, _ = conn.Write(resultStored)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultStored)))
	}
}

// delete() handles memcached `delete` command.
func (m *MiniMemcached) delete(key string, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	if item := m.items[key]; item == nil {
		_, _ = conn.Write(resultNotFound)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotFound)))
		}
		return
	}

	delete(m.items, key)

	_, _ = conn.Write(resultDeleted)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultDeleted)))
	}
}

// incr() handles memcached `incr` command.
func (m *MiniMemcached) incr(key string, incrValue uint64, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	item := m.items[key]
	if item == nil {
		_, _ = conn.Write(resultNotFound)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotFound)))
		}
		return
	}

	numericItemValue, isNumeric := getNumericValueFromByteArray(item.value)
	if !isNumeric {
		_, _ = conn.Write(resultClientErrIncrDecrNonNumericValue)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrIncrDecrNonNumericValue)))
		}
		return
	}

	item.casToken = m.incrementCASToken()

	var (
		numericItemValueInt big.Int
		numericIncrValueInt big.Int
		incrementedValueInt big.Int

		incrementedValue uint64
	)

	numericItemValueInt.SetUint64(numericItemValue)
	numericIncrValueInt.SetUint64(incrValue)
	incrementedValueInt.Add(&numericItemValueInt, &numericIncrValueInt)

	if incrementedValueInt.IsUint64() {
		incrementedValue = numericItemValue + incrValue
	} else {
		incrementedValue = 0
	}

	value := []byte(strconv.FormatUint(incrementedValue, 10))
	item.value = value
	result := append(value, crlf...)

	_, _ = conn.Write(result)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(result)))
	}
}

// decr() handles memcached `decr` command.
func (m *MiniMemcached) decr(key string, decrValue uint64, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	item := m.items[key]
	if item == nil {
		_, _ = conn.Write(resultNotFound)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotFound)))
		}
		return
	}
	numericItemValue, isNumeric := getNumericValueFromByteArray(item.value)
	if !isNumeric {
		_, _ = conn.Write(resultClientErrIncrDecrNonNumericValue)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrIncrDecrNonNumericValue)))
		}
		return
	}

	item.casToken = m.incrementCASToken()

	var decrementedValue uint64
	if numericItemValue < decrValue {
		decrementedValue = 0
	} else {
		decrementedValue = numericItemValue - decrValue
	}
	value := []byte(strconv.FormatUint(decrementedValue, 10))
	item.value = value
	result := append(value, crlf...)

	_, _ = conn.Write(result)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(result)))
	}
}

// touch() handles memcached `touch` command.
func (m *MiniMemcached) touch(key string, expiration int32, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	item := m.items[key]
	if item == nil {
		_, _ = conn.Write(resultNotFound)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotFound)))
		}
		return
	}
	item.expiration = expiration

	_, _ = conn.Write(resultTouched)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultTouched)))
	}
}

// flushAll() handles memcached `flush_all` command.
func (m *MiniMemcached) flushAll(conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = m.incrementCASToken()
	m.items = map[string]*item{}

	_, _ = conn.Write(resultOK)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultOK)))
	}
}

// cas() handles memcached `cas` command.
func (m *MiniMemcached) cas(key string, item *item, bytes int, casToken uint64, conn net.Conn) {
	if !isLegalKey(key) {
		_, _ = conn.Write(resultClientErrBadCliFormat)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadCliFormat)))
		}
		return
	}
	if !isLegalValue(bytes, item.value) {
		_, _ = conn.Write(resultClientErrBadDataChunk)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultClientErrBadDataChunk)))
		}
		return
	}

	m.invalidate(key)

	m.mu.Lock()
	defer m.mu.Unlock()
	prevItem := m.items[key]
	if prevItem == nil {
		_, _ = conn.Write(resultNotFound)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultNotFound)))
		}
		return
	}
	if prevItem.casToken != casToken {
		_, _ = conn.Write(resultExists)
		if m.logger.Level == Debug {
			m.logger.Println(fmt.Sprintf("result: %s", string(resultExists)))
		}
		return
	}

	item.casToken = m.incrementCASToken()
	m.items[key] = item

	_, _ = conn.Write(resultStored)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultStored)))
	}
}

// version() handles memcached `version` command,
func (m *MiniMemcached) version(conn net.Conn) {
	_, _ = conn.Write(resultVersion)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(resultVersion)))
	}
}

// handleErr() returns error response to client.
func (m *MiniMemcached) handleErr(err []byte, conn net.Conn) {
	_, _ = conn.Write(err)
	if m.logger.Level == Debug {
		m.logger.Println(fmt.Sprintf("result: %s", string(err)))
	}
}
