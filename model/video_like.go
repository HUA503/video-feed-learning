package model

import "time"

// 点赞
type VideoLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_video" json:"user_id"`
	VideoID   uint      `gorm:"not null;uniqueIndex:idx_user_video;index" json:"video_id"`
	CreatedAt time.Time `json:"created_at"`
}
