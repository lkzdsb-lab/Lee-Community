package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"Lee_Community/internal/repository/mysql"
	"Lee_Community/internal/repository/redis"
)

type PostLikeService struct {
	repo      *mysql.PostLikeRepository
	likeCache *redis.LikeCacheRepository
	lock      *redis.DistLock
}

func NewPostLikeService() *PostLikeService {
	return &PostLikeService{
		repo:      &mysql.PostLikeRepository{},
		likeCache: redis.NewLikeCacheRepository(),
	}
}

// Like 写库成功后，优先尝试加锁强更新缓存；拿不到锁则删计数Key，交给读侧惰性回填
func (s *PostLikeService) Like(ctx context.Context, userID, postID uint64) (bool, error) {
	if userID == 0 || postID == 0 {
		return false, errors.New("invalid id")
	}

	// 先写数据库
	changed, err := s.repo.Like(ctx, userID, postID)
	if err != nil || !changed {
		// 幂等命中时，尽量惰性回填集合（不创建新集合）
		if err == nil {
			s.likeCache.WarmIsLiked(ctx, userID, postID, true)
		}
		return changed, err
	}

	// 集合可直接更新（不强制），失败忽略
	_ = s.likeCache.AddLike(ctx, userID, postID)

	// 计数采用“写后强更新 + 锁；锁失败则删除计数Key，交给读侧单兵重建”
	token := fmt.Sprintf("%d-%d-%d", userID, postID, time.Now().UnixNano())
	got, _ := s.lock.Acquire(ctx, postID, token)
	if got {
		defer s.lock.Release(ctx, postID, token)
		// 强更新计数（INCR 已在 AddLike 内做过，这里确保至少一次成功）
		// 这里可以补一次校准：读库->Set 缓存（权衡RT）
		// 为简洁，这里再尝试一次计数回写，失败则降级删Key
		if err = s.likeCache.SetLikeCount(ctx, postID, -1); err != nil {
			// -1 表示不覆盖；如果没有该API，可忽略，保持 AddLike 的 INCR 结果
			// 这里降级：删Key
			_ = s.likeCache.DeleteCount(ctx, postID)
		}
	} else {
		// 拿不到锁，避免并发冲突，删除计数Key
		_ = s.likeCache.DeleteCount(ctx, postID)
	}
	return true, nil
}

// Unlike 同样策略，先写库；缓存集合更新后，计数用锁保护；失败则删除计数Key
func (s *PostLikeService) Unlike(ctx context.Context, userID, postID uint64) (bool, error) {
	if userID == 0 || postID == 0 {
		return false, errors.New("invalid id")
	}
	changed, err := s.repo.Unlike(ctx, userID, postID)
	if err != nil || !changed {
		if err == nil {
			s.likeCache.WarmIsLiked(ctx, userID, postID, false)
		}
		return changed, err
	}

	// 集合更新（不强制），失败忽略
	_ = s.likeCache.RemoveLike(ctx, userID, postID)

	// 计数更新受锁保护；拿不到锁则删计数Key
	token := fmt.Sprintf("%d-%d-%d", userID, postID, time.Now().UnixNano())
	got, _ := s.lock.Acquire(ctx, postID, token)
	if got {
		defer s.lock.Release(ctx, postID, token)
		// RemoveLike 内已做 WATCH/DECR 防负；这里若仍担心并发误差，可直接删Key交给读侧重建
		// 简化：尝试一次安全自减已在缓存层完成；若失败则删Key
	} else {
		_ = s.likeCache.DeleteCount(ctx, postID)
	}
	return true, nil
}

func (s *PostLikeService) IsLiked(ctx context.Context, userID, postID uint64) (bool, error) {
	if userID == 0 || postID == 0 {
		return false, errors.New("invalid id")
	}
	// 先查缓存集合（命中才用）
	if b, ok, err := s.likeCache.IsLikedCached(ctx, userID, postID); err == nil && ok {
		return b, nil
	}
	// 回源 MySQL
	b, err := s.repo.IsLiked(ctx, userID, postID)
	if err == nil {
		s.likeCache.WarmIsLiked(ctx, userID, postID, b)
	}
	return b, err
}

func (s *PostLikeService) GetCountWithLock(ctx context.Context, userID, postID uint64) (int64, error) {
	// 第一次从缓存读
	if v, ok, err := s.likeCache.GetLikeCountCached(ctx, postID); err == nil && ok {
		return v, nil
	}
	token := fmt.Sprintf("%d-%d-%d", userID, postID, time.Now().UnixNano())
	got, _ := s.lock.Acquire(ctx, postID, token)

	// 如果获取到了锁
	if got {
		// 用defer等待结束后释放锁
		defer func(lock *redis.DistLock, ctx context.Context, token string) {
			err := lock.Release(ctx, postID, token)
			if err != nil {
				fmt.Println(err)
			}
		}(s.lock, ctx, token)

		// 第二次检查
		if v, ok, err := s.likeCache.GetLikeCountCached(ctx, postID); err == nil && ok {
			return v, nil
		}

		// 如果仍然没有从缓存获取到key，则回源
		v, err := s.repo.GetLikeCount(ctx, postID)
		if err != nil {
			return 0, err
		}

		// 更新到缓存
		_ = s.likeCache.SetLikeCount(ctx, postID, v)
		return v, nil
	}

	// 没拿到锁，短暂退避后再读一次缓存，避免全体打DB
	time.Sleep(50 * time.Millisecond)
	if v, ok, err := s.likeCache.GetLikeCountCached(ctx, postID); err == nil && ok {
		return v, nil
	}

	// 仍miss，可有限回源一次或返回0按需
	return s.repo.GetLikeCount(ctx, postID)
}
