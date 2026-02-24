package service

import (
	"Lee_Community/internal/model"
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"Lee_Community/internal/pkg"
	"Lee_Community/internal/repository/mysql"
)

type FollowService struct {
	repo      *mysql.FollowRepository
	vipMarker *VIPMarker
}

// FollowCountReconciler 用户关注对账计数器
type FollowCountReconciler struct {
	repo      *mysql.FollowCountReconcilerRepo
	batchSize int
	interval  time.Duration
	lastID    uint64
}

type Sender func(ctx context.Context, ob *model.SocialOutbox) error

// OutboxRelayer outbox表相关服务
type OutboxRelayer struct {
	repo      *mysql.OutboxRepository
	batchSize int
	interval  time.Duration
	sender    func(ctx context.Context, ob *model.SocialOutbox) error
}

// VIPMarker 大v用户标记器
type VIPMarker struct {
	repo         *mysql.VIPMarkerRepo
	vipThreshold int64
}

func NewFollowService(vipThreshold int64) *FollowService {
	return &FollowService{
		repo:      &mysql.FollowRepository{},
		vipMarker: NewVIPMarker(vipThreshold),
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
		lastID:    0,               // 上一次对账的最后一个ID
	}
}

func NewVIPMarker(vipThreshold int64) *VIPMarker {
	return &VIPMarker{
		repo:         &mysql.VIPMarkerRepo{},
		vipThreshold: vipThreshold,
	}
}

func (s *FollowService) Follow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 {
		return false, errors.New("invalid user id")
	}
	if followerID == followeeID {
		return false, errors.New("cannot follow self")
	}
	// 幂等，仅仅返回是否修改
	changed, err := s.repo.Follow(ctx, followerID, followeeID)
	if err != nil {
		return false, err
	}
	// 只有关系从未关注->关注时才可能改变粉丝数阈值，changed=true
	if changed && s.vipMarker != nil {
		// 关注影响 followee 的 FollowerCount
		_ = s.vipMarker.CheckAndMark(ctx, followeeID)
	}
	return changed, nil
}

func (s *FollowService) Unfollow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 {
		return false, errors.New("invalid user id")
	}
	if followerID == followeeID {
		return false, errors.New("cannot unfollow self")
	}
	changed, err := s.repo.Unfollow(ctx, followerID, followeeID)
	if err != nil {
		return false, err
	}
	// 取关影响 followee 的 FollowerCount
	if changed && s.vipMarker != nil {
		_ = s.vipMarker.CheckAndMark(ctx, followeeID)
	}
	return changed, nil
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

// KafkaSender 构造一个将 outbox 事件发往 Kafka 的 sender
func KafkaSender(prod *pkg.KafkaProducer) func(ctx context.Context, ob *model.SocialOutbox) error {
	return func(ctx context.Context, ob *model.SocialOutbox) error {
		// 组装消息：沿用 Outbox 的字段
		payload := map[string]any{
			"event_type": ob.EventType,
			"follower":   ob.Follower,
			"followee":   ob.Followee,
			// Outbox.Payload 已含 event_time，可复用；也可解包合并，这里直接打包
			"payload":  ob.Payload,
			"event_id": ob.ID,
		}
		data, _ := json.Marshal(payload)
		key := pkg.MakeKeyFromID(ob.ID)
		if err := prod.Send(ctx, key, data); err != nil {
			return err
		}
		return nil
	}
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
	users, next, err := r.repo.ReconcileList(ctx, r.batchSize, r.lastID)
	if err != nil {
		log.Printf("reconcile list err: %v", err)
		return
	}
	// 推进游标：有数据则前进到本批最后一个 ID；无数据则重置为 0，下一轮从头开始
	if len(users) == 0 {
		r.lastID = 0
		return
	}
	r.lastID = next

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

// CheckAndMark 根据粉丝数动态设置 IsVIP，幂等更新
func (m *VIPMarker) CheckAndMark(ctx context.Context, userID uint64) error {
	var u model.User
	u, err := m.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	// 检查是否具有资格成为大V，如果已经是则直接返回
	want := u.FollowerCount >= m.vipThreshold
	if want == u.IsVIP {
		return nil
	}
	// 更新用户信息
	if err = m.repo.UpdateUser(ctx, u.ID, want); err != nil {
		return err
	}
	log.Printf("VIP marker: user=%d is_vip=%v (followers=%d, threshold=%d)", userID, want, u.FollowerCount, m.vipThreshold)
	return nil
}
