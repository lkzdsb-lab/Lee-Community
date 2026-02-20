package mysql

import (
	"Lee_Community/internal/model"

	"gorm.io/gorm"
)

type CommunityRepository struct {
	DB *gorm.DB
}

// Create 幂等地让创建者加入（角色=1）
func (r *CommunityRepository) Create(c *model.Community) (*model.Community, error) {
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		mRepo := &CommunityMemberRepository{DB: tx}

		if err := r.DB.Create(c).Error; err != nil {
			return err
		}

		// 幂等加入：仓储已 DoNothing；这里将其视为成功
		if err := mRepo.Join(&model.CommunityMember{
			CommunityID: c.ID,
			UserID:      c.CreatorID,
			Role:        1,
		}); err != nil {
			return err
		}

		return nil
	})
	return c, err
}

func (r *CommunityRepository) FindByID(id uint64) (*model.Community, error) {
	var community model.Community
	err := r.DB.First(&community, id).Error
	return &community, err
}

func (r *CommunityRepository) FindByName(name string) (*model.Community, error) {
	var community model.Community
	err := r.DB.Where("name = ?", name).First(&community).Error
	return &community, err
}

func (r *CommunityRepository) List(offset, limit int) ([]model.Community, error) {
	var list []model.Community
	err := r.DB.Order("id desc").Offset(offset).Limit(limit).Find(&list).Error
	return list, err
}

func (r *CommunityRepository) DeleteById(id uint64) error {
	// 幂等硬删除：无论是否存在，最终都视为成功
	tx := r.DB.Delete(&model.Community{}, id)
	if tx.Error != nil {
		return tx.Error
	}
	// 即使 RowsAffected == 0（已不存在），也返回 nil，保证幂等
	return nil
}
