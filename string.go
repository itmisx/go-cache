package cache

import (
	"time"

	"github.com/itmisx/timewheel"
)

// Set Key value with expiration and expiration callback function
func Set(key string, value interface{}, expiration time.Duration, expirationFunc func(key string, value interface{})) (success bool) {
	if key == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
	if expiration > 0 {
		c.itemFunc[key] = expirationFunc
		timewheel.AddTimer(key+":::"+"", expiration, map[string]string{
			"key":   key,
			"field": "",
		})
	}
	return true
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
