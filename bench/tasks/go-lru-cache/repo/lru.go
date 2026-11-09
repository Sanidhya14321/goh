package task

import "container/list"

type entry struct {
	key   string
	value int
}

// LRU is a fixed-capacity least-recently-used cache.
type LRU struct {
	cap   int
	order *list.List
	items map[string]*list.Element
}

func NewLRU(capacity int) *LRU {
	return &LRU{cap: capacity, order: list.New(), items: map[string]*list.Element{}}
}

func (c *LRU) Get(key string) (int, bool) {
	el, ok := c.items[key]
	if !ok {
		return 0, false
	}
	return el.Value.(*entry).value, true
}

func (c *LRU) Put(key string, value int) {
	el := c.order.PushFront(&entry{key, value})
	c.items[key] = el
	if c.order.Len() > c.cap {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*entry).key)
	}
}

func (c *LRU) Len() int { return c.order.Len() }
