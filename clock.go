package minimemcached

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

func NewClock() Clock {
	return &DefaultClock{}
}

type DefaultClock struct{}

func (c *DefaultClock) Now() time.Time {
	return time.Now()
}

func NewMockClock() *MockClock {
	return &MockClock{
		timeCursor: time.Now(),
	}
}

type MockClock struct {
	timeCursor time.Time
	lock       sync.Mutex
}

func (c *MockClock) Add(d time.Duration) {
	c.lock.Lock()
	c.timeCursor = c.timeCursor.Add(d)
	c.lock.Unlock()
}

func (c *MockClock) Now() time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.timeCursor
}
