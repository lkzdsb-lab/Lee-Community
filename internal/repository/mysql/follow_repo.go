package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"Lee_Community/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FollowRepository struct {
	DB *gorm.DB
}

type OutboxRepository struct {
	DB *gorm.DB
}

type FollowCountReconcilerRepo struct {
	DB *gorm.DB
}

type VIPMarkerRepo struct {
	DB *gorm.DB
}

// Pair 对账消息结构体
type Pair struct {
	ID             uint64
	FollowingCount int64
	FollowerCount  int64
}

// Follow 设置关系为关注（幂等）。如果状态从未关注切换为已关注，则返回 changed=true。
func (r *FollowRepository) Follow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	var changed bool
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rel model.Follow
		// select for update 避免竞争
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("follower_id=? AND followee_id=?", followerID, followeeID).First(&rel).Error; err != nil {
			// 如果没找到信息则创建
			if errors.Is(err, gorm.ErrRecordNotFound) {
				rel = model.Follow{
					FollowerID: followerID,
					FolloweeID: followeeID,
					Status:     1,
				}
				if err = tx.Create(&rel).Error; err != nil {
					return err
				}
				changed = true
				if err = r.adjustCounts(tx, followerID, followeeID, +1); err != nil {
					return err
				}
				// 写outbox表
				return r.insertOutbox(tx, "follow", followerID, followeeID)
			}
			return err
		}
		// 做幂等，判断是否真的是新关注还是重复请求，处理已经有关注信息的情况
		if rel.Status == 1 {
			changed = false
			return nil
		}
		if err := tx.Model(&model.Follow{}).
			Where("id=? AND status=0", rel.ID).
			Update("status", 1).Error; err != nil {
			return err
		}
		changed = true
		if err := r.adjustCounts(tx, followerID, followeeID, +1); err != nil {
			return err
		}

		return r.insertOutbox(tx, "follow", followerID, followeeID)
	})
	return changed, err
}

// Unfollow 处理粉丝关系
func (r *FollowRepository) Unfollow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	var changed bool
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rel model.Follow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("follower_id=? AND followee_id=?", followerID, followeeID).First(&rel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				changed = false
				return nil
			}
			return err
		}
		if rel.Status == 0 {
			changed = false
			return nil
		}
		if err := tx.Model(&model.Follow{}).
			Where("id=? AND status=1", rel.ID).
			Update("status", 0).Error; err != nil {
			return err
		}
		changed = true
		if err := r.adjustCounts(tx, followerID, followeeID, -1); err != nil {
			return err
		}

		return r.insertOutbox(tx, "unfollow", followerID, followeeID)
	})
	return changed, err
}

// IsFollowing 判断是否关注
func (r *FollowRepository) IsFollowing(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	var n int64
	if err := r.DB.WithContext(ctx).
		Model(&model.Follow{}).
		Where("follower_id=? AND followee_id=? AND status=1", followerID, followeeID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListFollowings 获取关注者列表
func (r *FollowRepository) ListFollowings(ctx context.Context, userID uint64, cursor uint64, limit int) ([]model.Follow, uint64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := r.DB.WithContext(ctx).Model(&model.Follow{}).
		Where("follower_id=? AND status=1", userID)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	var rows []model.Follow
	// 这里limit+1是为了更好的继续分页
	if err := q.Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	var next uint64
	if len(rows) > limit {
		next = rows[limit-1].ID
		rows = rows[:limit]
	}
	return rows, next, nil
}

// ListFollowers 获取粉丝列表
func (r *FollowRepository) ListFollowers(ctx context.Context, userID uint64, cursor uint64, limit int) ([]model.Follow, uint64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := r.DB.WithContext(ctx).Model(&model.Follow{}).
		Where("followee_id=? AND status=1", userID)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	var rows []model.Follow
	if err := q.Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	var next uint64
	if len(rows) > limit {
		next = rows[limit-1].ID
		rows = rows[:limit]
	}
	return rows, next, nil
}

// adjustCounts 自动调整关注这或粉丝数量
func (r *FollowRepository) adjustCounts(tx *gorm.DB, followerID, followeeID uint64, delta int64) error {
	if err := tx.Model(&model.User{}).
		Where("id=?", followerID).
		UpdateColumn("following_count", gorm.Expr("GREATEST(0, following_count + ?)", delta)).Error; err != nil {
		return err
	}
	if err := tx.Model(&model.User{}).
		Where("id=?", followeeID).
		UpdateColumn("follower_count", gorm.Expr("GREATEST(0, follower_count + ?)", delta)).Error; err != nil {
		return err
	}
	return nil
}

// 插入outbox事件表
func (r *FollowRepository) insertOutbox(tx *gorm.DB, event string, follower, followee uint64) error {
	payload, _ := json.Marshal(map[string]any{
		"event_time": time.Now().UTC().Format(time.RFC3339Nano),
		"follower":   follower,
		"followee":   followee,
	})
	ob := &model.SocialOutbox{
		EventType: event,
		Follower:  follower,
		Followee:  followee,
		Payload:   string(payload),
		Status:    0,
	}
	return tx.Create(ob).Error
}

// List outbox查询
func (r *OutboxRepository) List(ctx context.Context, batchSize int) ([]model.SocialOutbox, error) {
	var list []model.SocialOutbox
	if err := r.DB.WithContext(ctx).
		Where("status=0").
		Order("id ASC").
		Limit(batchSize).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// RetryUpdate outbox记录消息失败重试
func (r *OutboxRepository) RetryUpdate(ctx context.Context, id uint64) error {
	return r.DB.WithContext(ctx).Model(&model.SocialOutbox{}).Where("id=?", id).
		Updates(map[string]any{"status": 2, "retry": gorm.Expr("retry + 1")}).Error
}

// SuccessUpdate outbox成功记录消息更新
func (r *OutboxRepository) SuccessUpdate(ctx context.Context, id uint64) error {
	return r.DB.WithContext(ctx).Model(&model.SocialOutbox{}).Where("id=?", id).
		Update("status", 1).Error
}

// ReconcileList 异步对账用户批量查询
func (r *FollowCountReconcilerRepo) ReconcileList(ctx context.Context, batchSize int, lastID uint64) ([]Pair, uint64, error) {
	var list []Pair
	if err := r.DB.WithContext(ctx).Model(&model.User{}).
		Select("id", "following_count", "follower_count").
		Where("id > ?", lastID).
		Order("id ASC").
		Limit(batchSize).
		Find(&list).Error; err != nil {
		return nil, lastID, err
	}
	if len(list) == 0 {
		// 如果结果为空
		return nil, lastID, nil
	}
	// 正常批次
	return list, list[len(list)-1].ID, nil
}

// RealFollowers 真实粉丝数量查询
func (r *FollowCountReconcilerRepo) RealFollowers(ctx context.Context, userID uint64) (int64, error) {
	var realFollowing int64
	if err := r.DB.WithContext(ctx).Model(&model.Follow{}).
		Where("follower_id=? AND status=1", userID).
		Select("followee_id").
		Count(&realFollowing).Error; err != nil {
		return 0, err
	}
	return realFollowing, nil
}

// RealFollowings 真实关注的人数量查询
func (r *FollowCountReconcilerRepo) RealFollowings(ctx context.Context, userID uint64) (int64, error) {
	var realFollowing int64
	if err := r.DB.WithContext(ctx).Model(&model.Follow{}).
		Where("followee_id=? AND status=1", userID).
		Select("follower_id").
		Count(&realFollowing).Error; err != nil {
		return 0, err
	}
	return realFollowing, nil
}

// ReconcileFollowers 修正粉丝数量
func (r *FollowCountReconcilerRepo) ReconcileFollowers(ctx context.Context, userID uint64, realFollowing int64) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).Where("id=?", userID).
		UpdateColumn("following_count", realFollowing).Error
}

// ReconcileFollowings 修正关注的人的数量
func (r *FollowCountReconcilerRepo) ReconcileFollowings(ctx context.Context, userID uint64, realFollower int64) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).Where("id=?", userID).
		UpdateColumn("follower_count", realFollower).Error
}

// GetUser 获取需要标记的用户
func (m *VIPMarkerRepo) GetUser(ctx context.Context, userID uint64) (model.User, error) {
	var user model.User
	if err := m.DB.WithContext(ctx).
		Select("id, ", "is_vip", "follower_count").
		Where("id=?", userID).
		First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

// UpdateUser 更新用户信息为大v用户
func (m *VIPMarkerRepo) UpdateUser(ctx context.Context, userID uint64, isVip bool) error {
	return m.DB.WithContext(ctx).Model(&model.User{}).Where("id=?", userID).
		Update("is_vip", isVip).Error
}
