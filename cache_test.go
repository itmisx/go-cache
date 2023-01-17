package cache

import (
	"log"
	"testing"
	"time"
)

func TestXxx(t *testing.T) {
	// string test
	for i := 0; i < 1; i++ {
		go func() {
			Set("key1", 1, time.Second*4, func(key string, value interface{}) {
				log.Println("callback1", key, value)
			})
			Expire("key1", time.Second*7)
		}()
	}
	for i := 0; i < 1; i++ {
		go func() {
			HSet("hkey1", "hfield1", 1, time.Second*8, func(key string, field string, value interface{}) {
				log.Println("callback2", key, field, value)
			})
			Expire("hkey1", time.Second*7)
		}()
	}
	log.Println(Del("afsdf"))
	log.Println(Get("key1"))
	log.Println(Get("key2"))
	log.Println(HDel("hkey1", "hfield1"))
	log.Println(HGet("hkey1", "hfield1"))
	log.Println(HGet("hkey1", "hfield2"))
	time.Sleep(time.Second * 10)
	for {
		time.Sleep(time.Second)
	}
}
