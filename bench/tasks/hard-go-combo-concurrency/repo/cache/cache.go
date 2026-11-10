package cache

import (
	"sync"
	"time"
)

// Clock lets tests control time.
type Clock func() time.Time

type entry struct {
	value   string
	expires time.Time
}

// TTL is a string cache whose entries expire after ttl.
type TTL struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   Clock
	items map[string]entry
}

func NewTTL(ttl time.Duration, now Clock) *TTL {
	return &TTL{ttl: ttl, now: now, items: map[string]entry{}}
}

func (c *TTL) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || !c.now().Before(e.expires) {
		delete(c.items, key)
		return "", false
	}
	return e.value, true
}

func (c *TTL) Set(key, value string) {
	c.mu.Lock()
	c.items[key] = entry{value, c.now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *TTL) Invalidate(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}
