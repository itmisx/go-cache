package cache

import (
	"sync"
	"time"

	"github.com/itmisx/timewheel"
)

var c = &cache{}

type cache struct {
	mu            sync.RWMutex
	items         map[string]interface{}
	itemFunc      map[string]func(key string, value interface{})
	itemFieldFunc map[string]map[string]func(key string, field string, value interface{})
}

// return
func init() {
	c.items = make(map[string]interface{})
	c.itemFunc = make(map[string]func(key string, value interface{}))
	c.itemFieldFunc = make(map[string]map[string]func(key string, field string, value interface{}))
	timewheel.Start(runJanitor)
}

// Reset the expiration
func Expire(key string, expiration time.Duration) (success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; !ok {
		return false
	}
	if expiration > 0 {
		timewheel.AddTimer(key+":::"+"", expiration, map[string]string{
			"key":   key,
			"field": "",
		})
	}
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

			// remove item field resource
			{
				if fieldMap, ok := c.items[key].(map[string]interface{}); ok {
					for field := range fieldMap {
						removeJanitor(key, field)
					}
				}
			}

			//  remove item resource
			{
				delete(c.items, key)
				removeJanitor(key, "")
			}
		}
	}
	return count
}

// Create a goroutine to clean the key resource when timer expired
func runJanitor(data interface{}) {
	dataMap := data.(map[string]string)
	key := dataMap["key"]
	field := dataMap["field"]
	if field == "" {
		var callback func(key string, value interface{})
		// Lock
		c.mu.Lock()
		callbackValue := c.items[key]
		// if exits expiration callback , exec it then delete
		// hashmap key has no expiration callback
		if _, ok := c.itemFunc[key]; ok && c.itemFunc[key] != nil {
			callback = c.itemFunc[key]
		}
		// delete item callback
		delete(c.itemFunc, key)

		// remove item
		delete(c.items, key)

		// release Lock
		c.mu.Unlock()

		// exec expiration callback at last,avoid deadlock
		if callback != nil {
			go callback(key, callbackValue)
		}
	} else {
		var callback func(key string, field string, value interface{})
		c.mu.Lock()
		// get item value
		val := c.items[key]
		fieldMap, _ := val.(map[string]interface{})
		callbackValue := fieldMap[field]
		// exec the expiration callback and then delete the callback
		// if field func's length equals zero, delete the whole field's funcs
		if c.itemFieldFunc[key][field] != nil {
			callback = c.itemFieldFunc[key][field]
		}
		delete(c.itemFieldFunc[key], field)
		if len(c.itemFieldFunc[key]) == 0 {
			delete(c.itemFieldFunc, key)
		}
		// delete the item field
		delete(fieldMap, field)
		c.items[key] = fieldMap
		// if field length equals zero , delete whole item
		if len(fieldMap) == 0 {
			delete(c.items, key)
		}
		c.mu.Unlock()
		// exec expiration callback at last,avoid deadlock
		if callback != nil {
			go callback(key, field, callbackValue)
		}
	}
}

// removeJanitor
// remove the janitor of the item or item field
func removeJanitor(key string, field string) {
	timewheel.StopTimer(key + ":::" + field)
	if field == "" {
		// remove the item func
		delete(c.itemFunc, key)
	} else {
		// remove item field expiration callback function
		if _, ok := c.itemFieldFunc[key]; ok {
			delete(c.itemFieldFunc[key], field)
			if len(c.itemFieldFunc[key]) == 0 {
				delete(c.itemFieldFunc, key)
			}
		}
	}
}
