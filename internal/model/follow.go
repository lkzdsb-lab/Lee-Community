package model

import "time"

type Follow struct {
	ID         uint64 `gorm:"primaryKey"`
	FollowerID uint64 `gorm:"not null;index:idx_follower_id"`
	FolloweeID uint64 `gorm:"not null;index:idx_followee_id"`
	Status     int8   `gorm:"not null;default:1;comment:'1=follow,0=unfollow'"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TableName sets table name for Follow
func (Follow) TableName() string {
	return "follow"
}

// SocialOutbox 关注事件监控表
type SocialOutbox struct {
	ID        uint64 `gorm:"primaryKey"`
	EventType string `gorm:"size:16;not null"` // follow / unfollow
	Follower  uint64 `gorm:"not null"`
	Followee  uint64 `gorm:"not null"`
	Payload   string `gorm:"type:json;not null"`
	Status    int8   `gorm:"not null;default:0;comment:'0=pending,1=sent,2=failed'"`
	Retry     int    `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (SocialOutbox) TableName() string { return "social_outbox" }
