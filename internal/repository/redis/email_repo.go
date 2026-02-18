package redis

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultEmailCodeTTL = 5 * time.Minute
	EmailCodePrefix     = "email:code:"
	EmailRegisterPrefix = EmailCodePrefix + "register"
	CodeResetPrefix     = EmailCodePrefix + "reset"

	// 两阶段键
	PendingSuffix   = "pending"
	ConfirmedSuffix = "confirmed"
)

var (
	ErrEmailAddFailed      = errors.New("email add failed")
	ErrEmailNotFound       = errors.New("email not found")
	ErrEmailCodeDelFailed  = errors.New("email code delete failed")
	ErrCodePendingFailed   = errors.New("code pending failed")
	ErrCodeConfirmedFailed = errors.New("code confirmed failed")
)

type EmailRepository struct{}

/*
注册相关方法
*/

func (e *EmailRepository) RegisterEmailCodePending(email, code string) error {
	key := fmt.Sprintf("%s:%s:%s", EmailRegisterPrefix, PendingSuffix, email)
	if err := Client.Set(context.Background(), key, code, DefaultEmailCodeTTL).Err(); err != nil {
		return ErrCodePendingFailed
	}
	return nil
}

func (e *EmailRepository) MarkRegisterCodePending(email string) error {
	srcKey := fmt.Sprintf("%s:%s:%s", EmailRegisterPrefix, PendingSuffix, email)
	dstKey := fmt.Sprintf("%s:%s:%s", EmailRegisterPrefix, ConfirmedSuffix, email)

	// 使用lua脚本原子执行：取值+写入目标+设置 TTL+删除源
	script := `
local val = redis.call("GET", KEYS[1])
if not val then
  return 0
end
redis.call("SET", KEYS[2], val, "PX", ARGV[1])
redis.call("DEL", KEYS[1])
return 1
`
	px := int64(DefaultEmailCodeTTL / time.Millisecond)
	res := Client.Eval(context.Background(), script, []string{srcKey, dstKey}, px)
	if err := res.Err(); err != nil {
		return ErrCodeConfirmedFailed
	}
	ok, _ := res.Int()
	if ok != 1 {
		return ErrCodeConfirmedFailed
	}
	return nil
}

// DeleteRegisterCodePending 删除 pending 键（幂等）
func (e *EmailRepository) DeleteRegisterCodePending(email string) error {
	key := fmt.Sprintf("%s:%s:%s", EmailRegisterPrefix, PendingSuffix, email)
	if err := Client.Del(context.Background(), key).Err(); err != nil {
		return ErrEmailCodeDelFailed
	}
	return nil
}

// GetRegisterConfirmed 获取 confirmed 的验证码（校验时使用）
func (e *EmailRepository) GetRegisterConfirmed(email string) (string, error) {
	key := fmt.Sprintf("%s:%s:%s", EmailRegisterPrefix, ConfirmedSuffix, email)
	val, err := Client.Get(context.Background(), key).Result()
	if err != nil {
		return "", ErrEmailNotFound
	}
	return val, nil
}

/*
校验相关方法
*/

// GetEmailCode verify时用
func (e *EmailRepository) GetEmailCode(scope, email string) (string, error) {
	key := fmt.Sprintf("%s:%s:%s:%s", EmailCodePrefix, scope, ConfirmedSuffix, email)
	if err := Client.Get(context.Background(), key).Err(); err != nil {
		return "", ErrEmailNotFound
	}
	return key, nil
}

func (e *EmailRepository) DeleteEmailCode(scope, email string) error {
	key := fmt.Sprintf("%s:%s:%s", EmailCodePrefix, scope, ConfirmedSuffix, email)
	if err := Client.Del(context.Background(), key).Err(); err != nil {
		return ErrEmailCodeDelFailed
	}
	return nil
}

/*
重置密码相关方法
*/

// ResetEmailCodePending 写入重置验证码的 pending 键
func (e *EmailRepository) ResetEmailCodePending(email, code string) error {
	key := fmt.Sprintf("%s:%s:%s", CodeResetPrefix, PendingSuffix, email)
	if err := Client.Set(context.Background(), key, code, DefaultEmailCodeTTL).Err(); err != nil {
		return ErrCodePendingFailed
	}
	return nil
}

// MarkCodePending 将 pending 转为 confirmed（保留/重置 TTL）
func (e *EmailRepository) MarkCodePending(email string) error {
	srcKey := fmt.Sprintf("%s:%s:%s", CodeResetPrefix, PendingSuffix, email)
	dstKey := fmt.Sprintf("%s:%s:%s", CodeResetPrefix, ConfirmedSuffix, email)

	// 使用lua脚本原子执行：取值+写入目标+设置 TTL+删除源
	script := `
local val = redis.call("GET", KEYS[1])
if not val then
  return 0
end
redis.call("SET", KEYS[2], val, "PX", ARGV[1])
redis.call("DEL", KEYS[1])
return 1
`
	px := int64(DefaultEmailCodeTTL / time.Millisecond)
	res := Client.Eval(context.Background(), script, []string{srcKey, dstKey}, px)
	if err := res.Err(); err != nil {
		return ErrCodeConfirmedFailed
	}
	ok, _ := res.Int()
	if ok != 1 {
		return ErrCodeConfirmedFailed
	}
	return nil
}

// DeleteCodePending 删除 pending 键（幂等）
func (e *EmailRepository) DeleteCodePending(email string) error {
	key := fmt.Sprintf("%s:%s:%s", CodeResetPrefix, PendingSuffix, email)
	if err := Client.Del(context.Background(), key).Err(); err != nil {
		return ErrEmailCodeDelFailed
	}
	return nil
}

// GetResetConfirmed 获取 confirmed 的验证码（校验时使用）
func (e *EmailRepository) GetResetConfirmed(email string) (string, error) {
	key := fmt.Sprintf("%s:%s:%s", CodeResetPrefix, ConfirmedSuffix, email)
	val, err := Client.Get(context.Background(), key).Result()
	if err != nil {
		return "", ErrEmailNotFound
	}
	return val, nil
}
