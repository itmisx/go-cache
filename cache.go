package cache

import (
	"sync"
	"time"
)

var c = &cache{}

type cache struct {
	mu                sync.RWMutex
	items             map[string]interface{}
	itemTimer         map[string]*time.Timer
	itemFunc          map[string]func(key string, value interface{})
	itemChannel       map[string]chan bool
	itemFieldTimer    map[string]map[string]*time.Timer
	itemFieldFunc     map[string]map[string]func(key string, field string, value interface{})
	itemFieldChannel  map[string]map[string]chan bool
	defaultExpiration time.Duration
}

// return
func init() {
	c.items = make(map[string]interface{})
	c.itemTimer = make(map[string]*time.Timer)
	c.itemFunc = make(map[string]func(key string, value interface{}))
	c.itemChannel = make(map[string]chan bool)
	c.itemFieldTimer = make(map[string]map[string]*time.Timer)
	c.itemFieldFunc = make(map[string]map[string]func(key string, field string, value interface{}))
	c.itemFieldChannel = make(map[string]map[string]chan bool)
	c.defaultExpiration = time.Hour * 1
}

// Set the default expiration
func SetDefaultExpiration(expiration time.Duration) {
	if expiration > 0 {
		c.defaultExpiration = expiration
	}
}

// Reset the expiration
func Expire(key string, expiration time.Duration) (success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; !ok {
		return false
	}
	runJanitor(key, "", expiration)
	return true
}

// Delete the item
// remove the item or item field resource
func Del(keys ...string) (success int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var count int64
	for _, key := range keys {
		if _, ok := c.items[key]; ok {
			// success counter
			count++
			//  remove item resource
			{
				delete(c.items, key)
				removeJanitor(key, "")
			}

			// remove item field resource
			{
				if fieldTimers, ok := c.itemFieldTimer[key]; ok {
					for field := range fieldTimers {
						removeJanitor(key, field)
					}
				}
			}
		}
	}
	return count
}

// Create a goroutine to clean the key resource when timer expired
func runJanitor(key string, field string, expiration time.Duration) {

	if key == "" {
		panic("key can not be a empty string")
	}

	if expiration == 0 {
		expiration = c.defaultExpiration
	}
	tm := time.NewTimer(expiration)

	if field == "" {

		// if old chan not nil , push true to the old chan to stop the old janitor
		if c.itemChannel[key] != nil {
			c.itemChannel[key] <- true
		}

		// init the item chan
		c.itemChannel[key] = make(chan bool, 1)
		itemChan := c.itemChannel[key]

		// stop the old timer
		if c.itemTimer[key] != nil {
			if !c.itemTimer[key].Stop() {
				if len(c.itemTimer[key].C) > 0 {
					<-c.itemTimer[key].C
				}
				c.itemTimer[key].Stop()
			}
		}

		// init the timer
		c.itemTimer[key] = tm

		// create a go routine
		go func() {
			select {
			case <-tm.C:
				// Lock
				c.mu.Lock()

				// stop the timer
				tm.Stop()

				delete(c.itemTimer, key)

				// if exits expiration callback , exec it then delete
				// hashmap key has no expiration callback
				if _, ok := c.itemFunc[key]; ok && c.itemFunc[key] != nil {
					c.itemFunc[key](key, c.items[key])
				}
				delete(c.itemFunc, key)

				// delete key chan
				delete(c.itemChannel, key)

				// if item key is a hashmap key , remove the item field janitor
				if fieldMap, ok := c.items[key].(map[string]interface{}); ok {
					for field := range fieldMap {
						if c.itemFieldFunc[key][field] != nil {
							c.itemFieldFunc[key][field](key, field, fieldMap[field])
						}
						removeJanitor(key, field)
					}
				}

				// remove item
				delete(c.items, key)

				// release Lock
				c.mu.Unlock()
			case <-itemChan:
				// Lock
				c.mu.Lock()
				// close the item chan
				close(itemChan)
				// release Lock
				c.mu.Unlock()
			}
		}()
	} else {
		// init the field timer
		// if item field timer not nil , stop the old timer
		if c.itemFieldTimer[key] == nil {
			c.itemFieldTimer[key] = make(map[string]*time.Timer)
		} else {
			if timer, ok := c.itemFieldTimer[key][field]; ok {
				if !timer.Stop() {
					if len(timer.C) > 0 {
						<-timer.C
					}
					timer.Stop()
				}
			}
		}
		c.itemFieldTimer[key][field] = tm

		// init the item field chan
		// if item field chan not nil , push true to the old chan to stop the old janitor
		if c.itemFieldChannel[key] == nil {
			c.itemFieldChannel[key] = make(map[string]chan bool)
		} else {
			if c.itemFieldChannel[key][field] != nil {
				c.itemFieldChannel[key][field] <- true
			}
		}

		c.itemFieldChannel[key][field] = make(chan bool, 1)
		itemFieldChan := c.itemFieldChannel[key][field]

		// create a goroutine
		go func() {
			select {
			case <-tm.C:
				c.mu.Lock()
				defer c.mu.Unlock()
				// stop the timer
				tm.Stop()

				// delete the field
				val := c.items[key]
				fieldMap, _ := val.(map[string]interface{})
				delete(fieldMap, field)

				// if field length equals zero , delete whole item
				if len(fieldMap) == 0 {
					delete(c.items, key)
				}

				// delete field timer
				// if  field timer's length equals zero, delete the whole item field's timers
				delete(c.itemFieldTimer[key], field)
				if len(c.itemFieldTimer[key]) == 0 {
					delete(c.itemFieldTimer, key)
				}

				// exec the expiration callback and then delete the callback
				// if field func's length equals zero, delete the whole field's funcs
				if c.itemFieldFunc[key][field] != nil {
					c.itemFieldFunc[key][field](key, field, fieldMap[field])
				}
				delete(c.itemFieldFunc[key], field)
				if len(c.itemFieldFunc[key]) == 0 {
					delete(c.itemFieldFunc, key)
				}

				// delte the field chan
				// if field chan's length equals zero, delete the whole  field's chans
				delete(c.itemFieldChannel[key], field)
				if len(c.itemFieldChannel[key]) == 0 {
					delete(c.itemFieldChannel, key)
				}
			case <-itemFieldChan:
				c.mu.Lock()
				close(itemFieldChan)
				c.mu.Unlock()
			}
		}()
	}
}

// removeJanitor
// remove the janitor of the item or item field
func removeJanitor(key string, field string) {

	// stop and remove the item timer
	if _, ok := c.itemTimer[key]; ok {
		tm := c.itemTimer[key]
		if !tm.Stop() {
			<-tm.C
			tm.Stop()
		}
		// remove the item timer
		delete(c.itemTimer, key)
	}

	// remove the item func
	delete(c.itemFunc, key)

	// remove the item janitor chan
	if _, ok := c.itemChannel[key]; ok {
		c.itemChannel[key] <- true
		delete(c.itemChannel, key)
	}

	// stop and remove the item field timer
	if _, ok := c.itemFieldTimer[key]; ok {
		if _, ok := c.itemFieldTimer[key][field]; ok {
			tm := c.itemFieldTimer[key][field]
			if !tm.Stop() {
				<-tm.C
				tm.Stop()
			}
			delete(c.itemFieldTimer[key], field)
		}
		if len(c.itemFieldTimer[key]) == 0 {
			delete(c.itemFieldTimer, key)
		}
	}

	// remove item field expiration callback function
	if _, ok := c.itemFieldFunc[key]; ok {
		delete(c.itemFieldFunc[key], field)
		if len(c.itemFieldFunc[key]) == 0 {
			delete(c.itemFieldFunc, key)
		}
	}

	// remove item field janitor chan
	if _, ok := c.itemFieldChannel[key]; ok {
		if _, ok := c.itemFieldChannel[key][field]; ok {
			c.itemFieldChannel[key][field] <- true
			delete(c.itemFieldChannel[key], field)
		}
		if len(c.itemFieldChannel[key]) == 0 {
			delete(c.itemFieldChannel, key)
		}
	}
}
