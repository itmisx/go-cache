package cache

import (
	"log"
	"sync"
	"time"
)

var c = &cache{}

type cache struct {
	mu                sync.RWMutex
	items             map[string]interface{}
	itemTimers        map[string]*time.Timer
	itemFunc          map[string]func()
	itemChannel       map[string]chan bool
	itemFieldTimers   map[string]map[string]*time.Timer
	itemFieldFunc     map[string]map[string]func()
	itemFieldChannel  map[string]map[string]chan bool
	defaultExpiration time.Duration
}

// return
func init() {
	c.items = make(map[string]interface{})
	c.itemTimers = make(map[string]*time.Timer)
	c.itemFunc = make(map[string]func())
	c.itemChannel = make(map[string]chan bool)
	c.itemFieldTimers = make(map[string]map[string]*time.Timer)
	c.itemFieldFunc = make(map[string]map[string]func())
	c.itemFieldChannel = make(map[string]map[string]chan bool)
	c.defaultExpiration = time.Hour * 1
}

// DumpCache print cahce's resource
func DumpCache() {
	log.Println(c)
}

// Set the default expiration
func SetDefaultExpiration(expiration time.Duration) {
	if expiration > 0 {
		c.defaultExpiration = expiration
	}
}

// If the key/value exist  return TRUE , otherwise return false
func Exist(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, ok := c.items[key]; ok {
		return true
	}
	return true
}

// Reset the expiration
func Expire(key string, expiration time.Duration) (success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.itemTimers[key]; ok {
		c.itemTimers[key].Stop()
		c.itemTimers[key] = time.NewTimer(expiration)
	}
	runJanitor(key, "", expiration)
	return false
}

// Delete the key
// Sync key deadline
func Del(keys ...string) (success int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var count int64
	for _, key := range keys {
		if _, ok := c.items[key]; ok {
			// success counter
			count++
			// delete item resource
			{
				// stop the timer
				if !c.itemTimers[key].Stop() {
					<-c.itemTimers[key].C
				}
				// delete the item
				delete(c.items, key)
				// delete the item expire callback
				delete(c.itemFunc, key)
				// delete the item timer
				delete(c.itemTimers, key)
				// push a signal to the item chan
				c.itemChannel[key] <- true
				// delete item chan
				delete(c.itemChannel, key)
			}

			// handle field resource
			{
				// delete item field timer then delete the whole key
				if fieldTimers, ok := c.itemFieldTimers[key]; ok {
					for _, timer := range fieldTimers {
						if !timer.Stop() {
							<-timer.C
						}
					}
					delete(c.itemFieldTimers, key)
				}
				// delete item field expire callback then delete the whole key
				if fieldFuncs, ok := c.itemFieldFunc[key]; ok {
					for field := range fieldFuncs {
						delete(fieldFuncs, field)
					}
					delete(c.itemFieldFunc, key)
				}
				// push a signal to the item field chan then delete the whole key
				if fieldChans, ok := c.itemFieldChannel[key]; ok {
					for field := range fieldChans {
						fieldChans[field] <- true
					}
					delete(c.itemFieldChannel, key)
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
		// init the item chan
		if c.itemChannel[key] == nil {
			c.itemChannel[key] = make(chan bool, 1)
		}

		// push true to the old chan and give a new chan if old chan exists
		if c.itemChannel[key] != nil {
			c.itemChannel[key] <- true
			c.itemChannel[key] = make(chan bool, 1)
		}

		// close the old timer if exists
		if timer, ok := c.itemTimers[key]; ok {
			if !timer.Stop() {
				<-timer.C
			}
		}
		// create a new timer
		c.itemTimers[key] = tm
		// get the new chan
		itemChan := c.itemChannel[key]
		// create a go routine
		go func() {

			select {
			case <-tm.C:
				// Lock
				c.mu.Lock()

				// stop the timer
				tm.Stop()

				// remove key and it's timer
				delete(c.items, key)
				delete(c.itemTimers, key)

				// exec callback func and delete callback map
				c.itemFunc[key]()
				delete(c.itemFunc, key)

				// delete key chan
				delete(c.itemChannel, key)

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
		// init the field chan
		if c.itemFieldChannel[key] == nil {
			c.itemFieldChannel[key] = make(map[string]chan bool)
		}
		if c.itemFieldChannel[key][field] == nil {
			c.itemFieldChannel[key][field] = make(chan bool, 1)
		}

		// init the field timer
		if c.itemFieldTimers[key] == nil {
			c.itemFieldTimers[key] = make(map[string]*time.Timer)
		} else if timer, ok := c.itemFieldTimers[key][field]; ok {
			// stop the old timer and create a new timer
			if !timer.Stop() {
				<-timer.C
			}
			// push true to the old chan and give new chan
			c.itemFieldChannel[key][field] <- true
			c.itemFieldChannel[key][field] = make(chan bool, 1)

		}
		// create a new timer
		c.itemFieldTimers[key][field] = tm

		// get the new chan
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
				// if item field timer's length equals zero, delete the whole item field's timers
				delete(c.itemFieldTimers[key], field)
				if len(c.itemFieldTimers[key]) == 0 {
					delete(c.itemFieldTimers, key)
				}

				// exec the expiration callback and then delete the callback
				// if field func's length equals zero, delete the whole field's funcs
				c.itemFieldFunc[key][field]()
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
				defer c.mu.Unlock()
				close(itemFieldChan)
			}
		}()
	}
}
