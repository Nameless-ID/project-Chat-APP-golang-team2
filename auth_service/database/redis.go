package database

import (
	"auth-service/config"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cacher struct {
	rdb    *redis.Client
	expiry time.Duration
	prefix string
}

func newRedisClient(url, password string, dbIndex int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     url,
		Password: password,
		DB:       dbIndex,
	})
}

func NewCacher(cfg config.Config, expiry int) Cacher {
	cache := Cacher{
		rdb:    newRedisClient(cfg.RedisConfig.Url, cfg.RedisConfig.Password, 0),
		expiry: time.Duration(expiry) * time.Second,
		prefix: cfg.RedisConfig.Prefix,
	}

	if err := cache.rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}

	// Menampilkan pesan sukses koneksi
	fmt.Println("success connected to Redis")

	return cache
}

func (c *Cacher) Push(name string, value []byte) error {
	return c.rdb.RPush(context.Background(), c.prefix+"_"+name, value).Err()
}

func (c *Cacher) Pop(name string) (string, error) {
	return c.rdb.LPop(context.Background(), c.prefix+"_"+name).Result()
}

func (c *Cacher) GetLength(name string) int64 {
	return c.rdb.LLen(context.Background(), c.prefix+"_"+name).Val()
}

func (c *Cacher) Set(name string, value string) error {
	return c.rdb.Set(context.Background(), c.prefix+"_"+name, value, c.expiry).Err()
}

func (c *Cacher) SaveToken(name string, value string) error {
	return c.rdb.Set(context.Background(), c.prefix+"_"+name, value, 24*time.Hour).Err()
}

func (c *Cacher) Get(name string) (string, error) {
	return c.rdb.Get(context.Background(), c.prefix+"_"+name).Result()
}

func (c *Cacher) Delete(name string) error {
	return c.rdb.Del(context.Background(), c.prefix+"_"+name).Err()
}

func (c *Cacher) DeleteByKey(key string) error {
	return c.rdb.Del(context.Background(), key).Err()
}

func (c *Cacher) PrintKeys() {
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.rdb.Scan(context.Background(), cursor, "", 0).Result()
		if err != nil {
			panic(err)
		}

		for _, key := range keys {
			fmt.Println("key", key)
		}

		if cursor == 0 { // no more keys
			break
		}
	}
}

func (c *Cacher) GetKeys() []string {
	var cursor uint64
	var result []string
	for {
		var keys []string
		var err error
		keys, cursor, err = c.rdb.Scan(context.Background(), cursor, "", 0).Result()
		if err != nil {
			panic(err)
		}

		result = append(result, keys...)

		if cursor == 0 { // no more keys
			break
		}
	}

	return result
}

func (c *Cacher) GetKeysByPattern(pattern string) []string {
	var cursor uint64
	var result []string
	for {
		var keys []string
		var err error
		keys, cursor, err = c.rdb.Scan(context.Background(), cursor, pattern, 0).Result()
		if err != nil {
			panic(err)
		}

		result = append(result, keys...)

		if cursor == 0 { // no more keys
			break
		}
	}

	return result
}

// Pub and Sub
func (c *Cacher) Publish(channelName string, message string) error {
	return c.rdb.Publish(context.Background(), channelName, message).Err()
}

func (c *Cacher) Subcribe(channelName string) (*redis.Message, error) {
	subscriber := c.rdb.Subscribe(context.Background(), channelName)
	message, err := subscriber.ReceiveMessage(context.Background())
	return message, err
}

// Hash
func (c *Cacher) HSet(key, field, value string) error {
	return c.rdb.HSet(context.Background(), key, field, value).Err()
}

func (c *Cacher) HGet(key, field string) (string, error) {
	return c.rdb.HGet(context.Background(), key, field).Result()
}

func (c *Cacher) HDel(key, field string) error {
	return c.rdb.HDel(context.Background(), key, field).Err()
}

func (c *Cacher) HExists(key, field string) (bool, error) {
	return c.rdb.HExists(context.Background(), key, field).Result()
}

// Set
func (c *Cacher) SAdd(name string, values ...string) error {
	return c.rdb.SAdd(context.Background(), c.prefix+"_"+name, values).Err()
}

func (c *Cacher) SIsMember(name, value string) (bool, error) {
	return c.rdb.SIsMember(context.Background(), c.prefix+"_"+name, value).Result()
}

func (c *Cacher) SetWithExpiration(name string, value string, expiration time.Duration) error {
	return c.rdb.Set(context.Background(), c.prefix+"_"+name, value, expiration).Err()
}
