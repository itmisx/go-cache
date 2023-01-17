package cache

import (
	"time"
)

// Set Key value with expiration and expiration callback function
func Set(key string, value interface{}, expiration time.Duration, expirationFunc func(key string, val interface{})) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
	c.itemFunc[key] = expirationFunc
	runJanitor(key, "", expiration)
}

// Get the vlaue of given key , if exist return true, or return false
func Get(key string) (value interface{}, found bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, ok := c.items[key]; ok {
		return c.items[key], true
	}
	return nil, false
}
