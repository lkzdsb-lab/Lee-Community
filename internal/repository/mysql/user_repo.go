package mysql

import (
	"Lee_Community/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func (r *UserRepository) Create(user *model.User) error {
	return DB.Create(user).Error
}

func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.DB.Where("username = ? OR email = ?", username, username).First(&user).Error
	return &user, err
}

func (r *UserRepository) FindByID(id uint64) (*model.User, error) {
	var user model.User
	err := r.DB.First(&user, id).Error
	return &user, err
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var usr model.User
	err := r.DB.Where("email = ?", email).First(&usr).Error
	return &usr, err
}

func (r *UserRepository) UpdatePassword(user *model.User, newPassword string) error {
	return r.DB.Model(user).Update("password", newPassword).Error
}
