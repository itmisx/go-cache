package cache

import (
	"time"

	"github.com/itmisx/timewheel"
)

// HMSet Batch Set the hash field
func HMSet(key string, values ...interface{}) (success bool) {
	if len(values) == 0 || len(values)%2 != 0 {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	kv := make(map[string]interface{})
	for len(values) > 0 {
		if field, ok := values[0].(string); !ok {
			return false
		} else {
			value := values[1]
			kv[field] = value
		}
		values = values[2:]
	}
	for field, value := range kv {
		if c.items[key] != nil {
			if fieldMap, ok := c.items[key].(map[string]interface{}); !ok {
				return false
			} else {
				fieldMap[field] = value
				c.items[key] = fieldMap
			}
		} else {
			c.items[key] = make(map[string]interface{})
			fieldMap := make(map[string]interface{})
			fieldMap[field] = value
			c.items[key] = fieldMap
		}
	}
	return true
}

// HSet Set the hash field with expiration and expiration callback function
func HSet(
	key string,
	field string,
	value interface{},
	expiration time.Duration,
	expirationFunc func(key string, field string, value interface{}),
) (success bool) {
	if key == "" || field == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// init item field
	// if not a hashmap, return false
	if c.items[key] != nil {
		if fieldMap, ok := c.items[key].(map[string]interface{}); !ok {
			return false
		} else {
			fieldMap[field] = value
			c.items[key] = fieldMap
		}
	} else {
		c.items[key] = make(map[string]interface{})
		fieldMap := make(map[string]interface{})
		fieldMap[field] = value
		c.items[key] = fieldMap
	}

	// init item field expiration callback function
	if c.itemFieldFunc[key] == nil {
		c.itemFieldFunc[key] = make(map[string]func(key string, field string, value interface{}))
	}

	if expiration > 0 {
		c.itemFieldFunc[key][field] = expirationFunc
		timewheel.AddTimer(key+":::"+field, expiration, map[string]string{
			"key":   key,
			"field": field,
		})
	}
	return true
}

// Get All Hash key value
func HGetAll(key string) (value map[string]interface{}, found bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// if not a map, return false
	if fieldMap, ok := c.items[key].(map[string]interface{}); ok {
		return fieldMap, true
	} else {
		return nil, false
	}
}

// Get hash field value, if found return true
func HGet(key string, field string) (value interface{}, found bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// if not a map, return false
	if fieldMap, ok := c.items[key].(map[string]interface{}); ok {
		if fieldMap == nil {
			return nil, false
		}
		if v, ok := fieldMap[field]; ok {
			return v, true
		}
	}
	return nil, false
}

// Delete Hash field, if success return true
func HDel(key string, fields ...string) (count int64, success bool) {
	if key == "" {
		return 0, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// if not a map, return false
	if fieldMap, ok := c.items[key].(map[string]interface{}); ok {
		for _, field := range fields {
			if field == "" {
				continue
			}
			if _, ok := fieldMap[field]; ok {
				count++
				delete(fieldMap, field)
				removeJanitor(key, field)
			}
		}
		// reset the item value
		if len(fieldMap) > 0 {
			c.items[key] = fieldMap
		} else {
			delete(c.items, key)
		}
		// return the success number
		if count > 0 {
			return count, true
		} else {
			return 0, false
		}
	} else {
		return 0, false
	}
}

// Reset the expiration of the hash field
func HExpire(key string, field string, expiration time.Duration) (success bool) {
	if key == "" || field == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// if not a hash , return false
	if fieldMap, ok := c.items[key].(map[string]interface{}); ok {
		if _, ok := fieldMap[field]; !ok {
			return false
		}
	} else {
		return false
	}
	if expiration > 0 {
		timewheel.AddTimer(key+":::"+field, expiration, map[string]string{
			"key":   key,
			"field": field,
		})
	}
	return true
}
