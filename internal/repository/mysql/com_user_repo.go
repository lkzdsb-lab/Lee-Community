package mysql

import (
	"Lee_Community/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommunityMemberRepository struct {
	DB *gorm.DB
}

func (r *CommunityMemberRepository) Join(member *model.CommunityMember) error {
	// 幂等插入：若已存在 (community_id, user_id) 则不报错
	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "community_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(member).Error
}

func (r *CommunityMemberRepository) Leave(communityID, userID uint64) error {
	return r.DB.Where("community_id = ? AND user_id = ?", communityID, userID).
		Delete(&model.CommunityMember{}).Error
}

func (r *CommunityMemberRepository) IsMember(communityID, userID uint64) (bool, error) {
	var count int64
	err := r.DB.Model(&model.CommunityMember{}).
		Where("community_id = ? AND user_id = ?", communityID, userID).
		Count(&count).Error
	return count > 0, err
}
