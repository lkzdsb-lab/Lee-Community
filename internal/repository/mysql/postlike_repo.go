package mysql

import (
	"Lee_Community/internal/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type PostLikeRepository struct {
	DB *gorm.DB
}

func (r *PostLikeRepository) Like(ctx context.Context, userID, postID uint64) (bool, error) {
	tx := r.DB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var pl model.PostLike
	// 唯一(user_id, post_id) 幂等插入
	err := tx.
		Where("user_id = ? AND post_id = ?", userID, postID).
		First(&pl).Error
	if err == nil {
		// 已存在，幂等
		tx.Rollback()
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return false, err
	}

	// 如果没有查询到则创建
	if err = tx.Create(&model.PostLike{UserID: userID, PostID: postID}).Error; err != nil {
		tx.Rollback()
		return false, err
	}
	// 如果有则更新帖子计数
	if err = tx.Model(&model.Post{}).
		Where("id = ?", postID).
		UpdateColumn("like_count", gorm.Expr("like_count + 1")).
		Error; err != nil {
		tx.Rollback()
		return false, err
	}
	return tx.Commit().Error == nil, tx.Commit().Error
}

func (r *PostLikeRepository) Unlike(ctx context.Context, userID, postID uint64) (bool, error) {
	tx := DB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	res := tx.Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.PostLike{})
	if res.Error != nil {
		tx.Rollback()
		return false, res.Error
	}
	// 未删除任何行 -> 幂等
	if res.RowsAffected == 0 {
		tx.Rollback()
		return false, nil
	}
	// 计数-1，防止负数由业务层或对账兜底
	if err := tx.Model(&model.Post{}).
		Where("id = ?", postID).
		UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).
		Error; err != nil {
		tx.Rollback()
		return false, err
	}
	return tx.Commit().Error == nil, tx.Commit().Error
}

func (r *PostLikeRepository) IsLiked(ctx context.Context, userID, postID uint64) (bool, error) {
	var count int64
	err := DB.WithContext(ctx).
		Model(&model.PostLike{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error
	return count > 0, err
}

func (r *PostLikeRepository) GetLikeCount(ctx context.Context, postID uint64) (int64, error) {
	var p model.Post
	err := DB.WithContext(ctx).Select("id", "like_count").First(&p, postID).Error
	if err != nil {
		return 0, err
	}
	return p.LikeCount, nil
}
