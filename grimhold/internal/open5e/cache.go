package open5e

import (
	"sync"
	"time"
)

type entry struct {
	body []byte
	exp  time.Time
}

// Cache is an in-memory TTL map. Get treats an expired entry as a miss.
// Reap deletes expired entries on a timer.
type Cache struct {
	mu  sync.Mutex
	ttl time.Duration
	now func() time.Time
	m   map[string]entry
}

func newCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl: ttl,
		now: time.Now,
		m:   make(map[string]entry),
	}
}

func (c *Cache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || !c.now().Before(e.exp) {
		return nil, false
	}
	out := make([]byte, len(e.body))
	copy(out, e.body)
	return out, true
}

func (c *Cache) set(key string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]byte, len(body))
	copy(cp, body)
	c.m[key] = entry{body: cp, exp: c.now().Add(c.ttl)}
}

func (c *Cache) purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for k, e := range c.m {
		if !now.Before(e.exp) {
			delete(c.m, k)
		}
	}
}
