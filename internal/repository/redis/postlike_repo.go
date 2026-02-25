package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	LikeSetTTL       = 24 * time.Hour
	LikeCntTTL       = 24 * time.Hour
	LockTTL          = 300 * time.Millisecond
	LikeSetKeyPrefix = "like:set:post"   // 存放某个帖子已点赞的用户ID集合
	LikeCntKeyPrefix = "like:cnt:post"   // 缓存某个帖子的点赞计数
	LockKeyPrefix    = "lock:like:post:" // 分布式锁
)

type LikeCacheRepository struct {
	// 可配置
	likeSetTTL time.Duration
	likeCntTTL time.Duration
}

type DistLock struct {
	RDB *redis.Client
}

func NewLikeCacheRepository() *LikeCacheRepository {
	return &LikeCacheRepository{
		likeSetTTL: LikeSetTTL,
		likeCntTTL: LikeCntTTL,
	}
}

func (r *LikeCacheRepository) likeSetKey(postID uint64) string {
	return fmt.Sprintf("%s:%d", LikeSetKeyPrefix, postID)
}
func (r *LikeCacheRepository) likeCntKey(postID uint64) string {
	return fmt.Sprintf("%s:%d", LikeCntKeyPrefix, postID)
}

// AddLike 写路径：成功写MySQL后再调用这些方法
func (r *LikeCacheRepository) AddLike(ctx context.Context, userID, postID uint64) error {
	// 写用户
	k := r.likeSetKey(postID)
	if err := Client.SAdd(ctx, k, userID).Err(); err != nil {
		return err
	}
	// 设置过期时间
	_ = Client.Expire(ctx, k, r.likeSetTTL).Err()

	// 写计数
	ck := r.likeCntKey(postID)
	if err := Client.Incr(ctx, ck).Err(); err != nil {
		return err
	}
	_ = Client.Expire(ctx, ck, r.likeCntTTL).Err()
	return nil
}

func (r *LikeCacheRepository) RemoveLike(ctx context.Context, userID, postID uint64) error {
	k := r.likeSetKey(postID)
	// 删除用户对帖子的点赞记录
	if err := Client.SRem(ctx, k, userID).Err(); err != nil {
		return err
	}
	ck := r.likeCntKey(postID)
	// 计数防负数
	if err := Client.Watch(ctx, func(tx *redis.Tx) error {
		val, err := tx.Get(ctx, ck).Int64()
		if err != nil && !errors.Is(err, redis.Nil) {
			return err
		}
		if val <= 0 {
			// 若不存在或<=0，直接返回，交给对账兜底
			return nil
		}
		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			p.Decr(ctx, ck)
			return nil
		})
		return err
	}, ck); err != nil {
		return err
	}
	return nil
}

// IsLikedCached 从缓存查看用户是否已经对帖子投过票了
func (r *LikeCacheRepository) IsLikedCached(ctx context.Context, userID, postID uint64) (bool, bool, error) {
	k := r.likeSetKey(postID)
	exists, err := Client.Exists(ctx, k).Result()
	if err != nil {
		return false, false, err
	}
	if exists == 0 {
		return false, false, nil
	}
	b, err := Client.SIsMember(ctx, k, userID).Result()
	return b, true, err
}

// GetLikeCountCached 从缓存读取帖子的点赞数量
func (r *LikeCacheRepository) GetLikeCountCached(ctx context.Context, postID uint64) (int64, bool, error) {
	ck := r.likeCntKey(postID)
	val, err := Client.Get(ctx, ck).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	return val, true, err
}

// SetLikeCount 回填帖子点赞数
func (r *LikeCacheRepository) SetLikeCount(ctx context.Context, postID uint64, cnt int64) error {
	ck := r.likeCntKey(postID)
	if err := Client.Set(ctx, ck, cnt, r.likeCntTTL).Err(); err != nil {
		return err
	}
	return nil
}

func (r *LikeCacheRepository) WarmIsLiked(ctx context.Context, userID, postID uint64, liked bool) {
	// 惰性回填：只在集合已存在时写，避免无界扩张
	// “无界扩张”指的是某个数据结构的体量在没有上限约束的情况下持续增长，最终可能占用过多内存或导致性能劣化
	// “惰性回填”的策略：
	// 只有当该集合已存在时才回填（Exists>0），否则不创建新集合；
	// 结合较长但有限的过期时间（TTL），让长期不访问的集合自动淘汰。
	k := r.likeSetKey(postID)
	if ok, _ := Client.Exists(ctx, k).Result(); ok > 0 {
		if liked {
			_ = Client.SAdd(ctx, k, userID).Err()
		} else {
			_ = Client.SRem(ctx, k, userID).Err()
		}
		_ = Client.Expire(ctx, k, r.likeSetTTL).Err()
	}
}

// DeleteCount 安全删除计数缓存，支持可选延迟二删，减少并发窗口脏数据
// 语义：立刻删除计数Key；若需要，可在delay>0时再异步删除一次
func (r *LikeCacheRepository) DeleteCount(ctx context.Context, postID uint64, delay ...time.Duration) error {
	key := r.likeCntKey(postID)
	// 第一次删除（立即）
	if err := Client.Del(ctx, key).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	// 可选的延迟二删
	if len(delay) > 0 && delay[0] > 0 {
		d := delay[0]
		// 轻量异步：在后台再删一次，抵消并发回填窗口
		go func() {
			t := time.NewTimer(d)
			defer t.Stop()
			<-t.C
			_ = Client.Del(context.Background(), key).Err()
		}()
	}
	return nil
}

// Acquire 请求加分布式锁
func (l *DistLock) Acquire(ctx context.Context, postID uint64, token string) (bool, error) {
	key := fmt.Sprintf("%s:%d", LockKeyPrefix, postID)
	return l.RDB.SetNX(ctx, key, token, LockTTL).Result()
}

// Release 用lua保证原子性
func (l *DistLock) Release(ctx context.Context, postID uint64, token string) error {
	key := fmt.Sprintf("%s:%d", LockKeyPrefix, postID)
	_, err := redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
else
  return 0
end`).Run(ctx, l.RDB, []string{key}, token).Result()
	return err
}
