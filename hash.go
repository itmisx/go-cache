package cache

import "time"

// Set the hash field with expiration and expiration callback function
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
	c.itemFieldFunc[key][field] = expirationFunc

	runJanitor(key, field, expiration)
	return true
}

// Get All Hash key value
func HGetALL(key string) (value map[string]interface{}, found bool) {
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
	c.mu.Lock()
	defer c.mu.Unlock()
	// if not a map, return false
	if fieldMap, ok := c.items[key].(map[string]interface{}); ok {
		for _, field := range fields {
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

	runJanitor(key, field, expiration)
	return true
}
