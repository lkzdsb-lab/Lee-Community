package model

import "time"

type PostLike struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `gorm:"index;not null"`
	PostID    uint64 `gorm:"index;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (PostLike) TableName() string {
	return "post_likes"
}
