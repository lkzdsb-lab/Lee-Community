package model

import "time"

type User struct {
	ID             uint64 `gorm:"primaryKey"`
	Username       string `gorm:"uniqueIndex;size:32;not null"`
	Password       string `gorm:"size:255;not null"`
	Role           int    `gorm:"default:0;not null;comment:'0=member, 1=admin'"`
	Email          string `gorm:"uniqueIndex;size:64;not null"`
	IsVIP          bool   `gorm:"default:false"` // 是否大V
	FollowerCount  int64  `gorm:"default:0"`
	FollowingCount int64  `gorm:"default:0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (User) TableName() string { return "user" }
