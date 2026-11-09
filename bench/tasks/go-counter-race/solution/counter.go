package task

import "sync"

// Counter counts events.
type Counter struct {
	mu sync.Mutex
	n  int
}

// Inc adds one.
func (c *Counter) Inc() { c.mu.Lock(); c.n++; c.mu.Unlock() }

// Value returns the count.
func (c *Counter) Value() int { c.mu.Lock(); defer c.mu.Unlock(); return c.n }
