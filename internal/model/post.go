package model

import "time"

type Post struct {
	ID          uint64    `gorm:"primaryKey; index:idx_comm_time_id,priority:3,sort:desc"`
	CommunityID uint64    `gorm:"not null;index:idx_community_time_id,priority:1"`
	AuthorID    uint64    `gorm:"not null;index:idx_author_time"`
	Title       string    `gorm:"size:200;not null"`
	Content     string    `gorm:"type:text"`
	Status      int       `gorm:"not null;default:0"` // 0=normal 1=deleted 2=banned
	LikeCount   int64     `gorm:"not null;default:0"`
	CreatedAt   time.Time `gorm:"index:idx_comm_time_id,priority:2,sort:desc, idx_author_time"`
	UpdatedAt   time.Time
}
