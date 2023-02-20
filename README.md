# go-cache

一个基于内存的 key/value 的 go 语言存储库，支持 string 和 hash, 类似redis

#### 🚀 安装

`go get -u github.com/itmisx/go-cache`

#### ✨ 特性

- 支持过期回调函数
- 支持哈希字段的过期时间设置

#### 🏗️ 使用场景

主要用在单机或本地缓存场景

#### ✅ 开始使用

- 设置键值，不带过期回调

```go
cache.Set("key1", 1, time.Second*4, nil)
```

- 设置键值，带过期回调

```go
cache.Set("key1", 1, time.Second*4, func(key string, value interface{}) {
    log.Println("callback1", key, value)
})
```

- 设置键的过期时间

```go
cache.Expire("key1",time.Second*4)
```

- 获取键值

```go
// found指示是否存在
value,found := cache.Get("key1")
```

- 删除键

```go
cache.Del("key1")
```

- 设置 hash，带过期回调

```go
cache.HSet("hkey1", "hfield1", 1, time.Second*8, func(key string, field string, value interface{}) {
    log.Println("callback2", key, field, value)
})
```

- 设置 hash 字段的过期时间

```go
cache.Expire("hkey1","hfield1",time.Second*3)
```

- 获取 hash 值

```go
// found指示是否存在
value,found := cache.HGet("hkey1","hfield1")
```

- 删除 hash 字段

```go
cache.HDel("hkey1","hfield1")
```
