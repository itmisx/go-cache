package cache

import (
	"log"
	"testing"
	"time"
)

// 2023/02/18 13:25:45 key: 1 true , hkey: 1 true
// 2023/02/18 13:25:46 key: 1 true , hkey: 1 true
// 2023/02/18 13:25:47 key: <nil> false , hkey: 1 true
// 2023/02/18 13:25:48 key: <nil> false , hkey: 1 true
// 2023/02/18 13:25:48 heky expired hkey field1 1
func TestXxx(t *testing.T) {
	end := false
	Set("key", 1, time.Second*2, nil)
	HSet("hkey", "field1", 1, time.Second*4, func(key, field string, value interface{}) {
		end = true
		log.Println("heky expired", key, field, value)
	})
	for {
		keyvalue, keyRes := Get("key")
		hkeyvalue, hkeyRes := HGet("hkey", "field1")
		time.Sleep(time.Second)
		log.Println("key:", keyvalue, keyRes, ",", "hkey:", hkeyvalue, hkeyRes)
		if end {
			break
		}
	}
}
