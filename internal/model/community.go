package model

import "time"

type Community struct {
	ID          uint64 `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex;size:64;not null"`
	Description string `gorm:"type:text"`
	CreatorID   uint64 `gorm:"not null;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CommunityMember struct {
	ID          uint64 `gorm:"primaryKey"`
	CommunityID uint64 `gorm:"not null;index;uniqueIndex:uk_community_user"`
	UserID      uint64 `gorm:"not null;index;uniqueIndex:uk_community_user"`
	Role        int    `gorm:"not null;default:0"` // 0=member, 1=admin
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
