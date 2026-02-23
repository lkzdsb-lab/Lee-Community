package service

import (
	"Lee_Community/internal/model"
	"context"
	"errors"
	"log"
	"time"

	"Lee_Community/internal/repository/mysql"
)

type FollowService struct {
	repo *mysql.FollowRepository
}

// FollowCountReconciler 用户关注对账计数器
type FollowCountReconciler struct {
	repo      *mysql.FollowCountReconcilerRepo
	batchSize int
	interval  time.Duration
}

type Sender func(ctx context.Context, ob *model.SocialOutbox) error

// OutboxRelayer outbox表相关服务
type OutboxRelayer struct {
	repo      *mysql.OutboxRepository
	batchSize int
	interval  time.Duration
	sender    func(ctx context.Context, ob *model.SocialOutbox) error
}

func NewFollowService() *FollowService {
	return &FollowService{
		repo: &mysql.FollowRepository{},
	}
}

func NewOutboxRelayer(sender Sender) *OutboxRelayer {
	return &OutboxRelayer{
		repo:      &mysql.OutboxRepository{},
		batchSize: 200,
		interval:  time.Second,
		sender:    sender,
	}
}

func NewFollowCountReconciler() *FollowCountReconciler {
	return &FollowCountReconciler{
		repo:      &mysql.FollowCountReconcilerRepo{},
		batchSize: 500,             // 设置一次对账的大小
		interval:  5 * time.Minute, // 对账的间隔时间
	}
}

func (s *FollowService) Follow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 {
		return false, errors.New("invalid user id")
	}
	if followerID == followeeID {
		return false, errors.New("cannot follow self")
	}
	return s.repo.Follow(ctx, followerID, followeeID)
}

func (s *FollowService) Unfollow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 {
		return false, errors.New("invalid user id")
	}
	if followerID == followeeID {
		return false, errors.New("cannot unfollow self")
	}
	return s.repo.Unfollow(ctx, followerID, followeeID)
}

func (s *FollowService) IsFollowing(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 {
		return false, errors.New("invalid user id")
	}
	return s.repo.IsFollowing(ctx, followerID, followeeID)
}

func (s *FollowService) ListFollowings(ctx context.Context, userID uint64, cursor uint64, limit int) (any, uint64, error) {
	return s.repo.ListFollowings(ctx, userID, cursor, limit)
}

func (s *FollowService) ListFollowers(ctx context.Context, userID uint64, cursor uint64, limit int) (any, uint64, error) {
	return s.repo.ListFollowers(ctx, userID, cursor, limit)
}

// Run outbox启动器
func (r *OutboxRelayer) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.drainOnce(ctx)
		}
	}
}

// Outbox 投递器，从数据库读取信息异步交给kafka传递消息
func (r *OutboxRelayer) drainOnce(ctx context.Context) {
	var rows []model.SocialOutbox
	// 按大小查询事件记录
	rows, err := r.repo.List(ctx, r.batchSize)
	if err != nil {
		log.Printf("outbox query err: %v", err)
		return
	}
	for i := range rows {
		ob := rows[i]
		if err = r.sender(ctx, &ob); err != nil {
			_ = r.repo.RetryUpdate(ctx, ob.ID).Error
			continue
		}
		_ = r.repo.SuccessUpdate(ctx, ob.ID).Error
	}
}

// LogSender 默认 sender（占位）：先打印，后续替换为 Kafka Producer
func LogSender(ctx context.Context, ob *model.SocialOutbox) error {
	log.Printf("OUTBOX SEND type=%s follower=%d followee=%d payload=%s", ob.EventType, ob.Follower, ob.Followee, ob.Payload)
	return nil
}

// ReconcilerRun 对账定时任务启动器
func (r *FollowCountReconciler) ReconcilerRun(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.reconcileOnce(ctx)
		}
	}
}

// 对账一次
func (r *FollowCountReconciler) reconcileOnce(ctx context.Context) {
	var users []mysql.Pair
	users, err := r.repo.ReconcileList(ctx, r.batchSize)
	if err != nil {
		log.Printf("reconcile list err: %v", err)
		return
	}

	for _, u := range users {
		// 先在follow表查询真实值，再和user表比对更新
		var realFollowing int64
		if realFollowing, err = r.repo.RealFollowings(ctx, u.ID); err != nil {
			continue
		}
		var realFollower int64
		if realFollower, err = r.repo.RealFollowers(ctx, u.ID); err != nil {
			continue
		}
		if realFollowing != u.FollowingCount {
			_ = r.repo.ReconcileFollowings(ctx, u.ID, realFollowing).Error()
		}
		if realFollower != u.FollowerCount {
			_ = r.repo.ReconcileFollowers(ctx, u.ID, realFollower).Error()
		}
	}
}
