package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrTokenNotFound    = errors.New("token not found")
	ErrTokenMismatch    = errors.New("token mismatch")
	ErrRedisUnavailable = errors.New("redis unavailable")
	ErrExtendFailed     = errors.New("token extend failed")
	ErrTokenDeleted     = errors.New("token delete failed")
)

const (
	UserTokenPrefix = "login:user:token"
	UserTokenExpire = 60 * 30
)

type UserRepository struct{} // 用户相关接口

func (r *UserRepository) AddUserToken(usrId uint64, token string) error {
	key := fmt.Sprintf("%s:%d", UserTokenPrefix, usrId)
	if err := Client.Set(context.Background(), key, token, time.Second*UserTokenExpire).Err(); err != nil {
		return ErrRedisUnavailable
	}
	return nil
}

func (r *UserRepository) GetUserToken(usrId uint64) (string, error) {
	key := fmt.Sprintf("%s:%d", UserTokenPrefix, usrId)
	token, err := Client.Get(context.Background(), key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrTokenNotFound
	}
	if err != nil {
		return "", ErrRedisUnavailable
	}
	return token, nil
}

func (r *UserRepository) ExtendUserToken(usrId uint64) error {
	key := fmt.Sprintf("%s:%d", UserTokenPrefix, usrId)
	_, err := Client.Expire(context.Background(), key, time.Second*UserTokenExpire).Result()
	if err != nil {
		return ErrExtendFailed
	}
	return nil
}

func (r *UserRepository) DeleteUserToken(usrId uint64) error {
	key := fmt.Sprintf("%s:%d", UserTokenPrefix, usrId)
	err := Client.Del(context.Background(), key).Err()
	if err != nil {
		return ErrTokenDeleted
	}
	return nil
}
