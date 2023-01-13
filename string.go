package cache

import (
	"time"
)

// Set Key value with expiration and expiration callback function
func Set(key string, value interface{}, expiration time.Duration, expirationFunc func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value

	// init item chan
	if c.itemChannel[key] == nil {
		c.itemChannel[key] = make(chan bool, 1)
	}
	// stop the old timer
	if _, ok := c.itemTimers[key]; ok {
		if !c.itemTimers[key].Stop() {
			<-c.itemTimers[key].C
		}
	}
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
