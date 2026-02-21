package mysql

import (
	"Lee_Community/internal/model"

	"gorm.io/gorm"
)

type PostRepository struct {
	DB *gorm.DB
}

func (r *PostRepository) Create(post *model.Post) error {
	return r.DB.Create(post).Error
}

func (r *PostRepository) FindByID(id uint64) (*model.Post, error) {
	var post model.Post
	err := DB.First(&post, "id = ? AND status = 0", id).Error
	return &post, err
}

// ListByCommunity 基础分页查询
func (r *PostRepository) ListByCommunity(communityID uint64, offset, limit int) ([]model.Post, error) {
	var list []model.Post
	err := DB.
		Where("community_id = ? AND status = 0", communityID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&list).Error
	return list, err
}

// ListByCommunityCursor 基于时间游标的查询：索引 (community_id, created_at DESC, id DESC)
// lastCreatedAt=零值表示第一页；否则用 (created_at, id) 作为严格游标
func (r *PostRepository) ListByCommunityCursor(communityID uint64, lastID uint64, lastCreatedAt int64, limit int) ([]model.Post, error) {
	var list []model.Post
	q := r.DB.Where("community_id = ? AND status = 0", communityID)
	if lastCreatedAt > 0 {
		// 标准时间游标：先比时间，再在同一时间点用 id 打破并列
		q = q.Where("(created_at < FROM_UNIXTIME(?) OR (created_at = FROM_UNIXTIME(?) AND id < ?))", lastCreatedAt, lastCreatedAt, lastID)
	}
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&list).Error
	return list, err
}

// Delete 软删除
func (r *PostRepository) Delete(id uint64) error {
	return DB.Model(&model.Post{}).
		Where("id = ?", id).
		Update("status", 1).Error
}

// DeleteWithPermission 带权限的一步删除：作者或管理员(role>=1)方可删除；幂等（已删除也不报错）
func (r *PostRepository) DeleteWithPermission(postID, operatorID uint64) (affected int64, err error) {
	tx := r.DB.Exec(`
		UPDATE posts p
		JOIN (SELECT id, community_id, author_id, status FROM posts WHERE id = ?) x ON x.id = p.id
		SET p.status = 1
		WHERE p.id = ? AND p.status = 0
		  AND (x.author_id = ? OR EXISTS (
		       SELECT 1 FROM community_members m
		       WHERE m.community_id = x.community_id AND m.user_id = ? AND m.role >= 1
		  ))`,
		postID, postID, operatorID, operatorID,
	)
	return tx.RowsAffected, tx.Error
}
