package config

import (
	"context"
	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     "192.168.215.3:6379",
		Password: "",
		DB:       0,
	})

	if err := RDB.Ping(context.Background()).Err(); err != nil {
		panic("redis connect error: " + err.Error())
	}
}
