package model

import "time"

type User struct {
	ID        uint64 `gorm:"primaryKey"`
	Username  string `gorm:"uniqueIndex;size:32;not null"`
	Password  string `gorm:"size:255;not null"`
	Role      int    `gorm:"default:0"`
	Email     string `gorm:"uniqueIndex;size:64;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
