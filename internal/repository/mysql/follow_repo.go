package mysql

import (
	"context"
	"errors"

	"Lee_Community/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FollowRepository struct {
	DB *gorm.DB
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
				if err := tx.Create(&rel).Error; err != nil {
					return err
				}
				changed = true
				return r.adjustCounts(tx, followerID, followeeID, +1)
			}
			return err
		}
		// 做幂等，判断是否真的是新关注还是重复请求
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
		return r.adjustCounts(tx, followerID, followeeID, +1)
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
		return r.adjustCounts(tx, followerID, followeeID, -1)
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
