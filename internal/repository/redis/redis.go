package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
)

// Init 初始化 Redis 客户端并做一次 Ping 健康检查。
func Init(addr, password string, db int) error {
	Client = redis.NewClient(&redis.Options{
		Addr:         addr,     // 例如 "127.0.0.1:6379"
		Password:     password, // 无密码则留空
		DB:           db,       // 使用的 DB 库号，默认 0
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     10, // 可按需求调整
		MinIdleConns: 2,  // 可按需求调整
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return Client.Ping(ctx).Err()
}

// Close 关闭 Redis 客户端（在程序退出时调用）。
func Close() error {
	if Client == nil {
		return nil
	}
	return Client.Close()
}
