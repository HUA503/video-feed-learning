package model

import "time"

type Video struct {
	ID          uint      `gorm:"primaryKey;index:idx_videos_user_created_id,priority:3,sort:desc" json:"id"`
	UserID      uint      `gorm:"not null;index;index:idx_videos_user_created_id,priority:1" json:"user_id"`
	Title       string    `gorm:"size:200;not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	VideoURL    string    `gorm:"size:500;not null" json:"video_url"`
	CoverURL    string    `gorm:"size:500" json:"cover_url"`
	CreatedAt   time.Time `gorm:"index:idx_videos_user_created_id,priority:2,sort:desc" json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
