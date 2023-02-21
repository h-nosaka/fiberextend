package fiberextend

import (
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

func (p *IFiberExConfig) NewRedis() *redis.Client {
	client := redis.NewClient(p.RedisOptions)
	if client == nil {
		panic("connection error: redis")
	}
	return client
}

// json型から変換して取得
func (p *IFiberEx) GetRedisJson(rs interface{}, key string) error {
	cmd := p.Redis.Get(background, key)
	if cmd.Err() != nil {
		return nil
	}
	value, _ := cmd.Result()
	if err := json.Unmarshal([]byte(value), &rs); err != nil {
		return err
	}
	return nil
}

// json型に変換して保存
func (p *IFiberEx) SetRedisJson(key string, src interface{}, expire time.Duration) error {
	value, err := json.Marshal(src)
	if err != nil {
		return err
	}
	if err := p.Redis.Set(background, key, value, expire).Err(); err != nil {
		return err
	}
	return nil
}
