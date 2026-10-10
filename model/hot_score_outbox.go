package model

import "time"

// 待发送事件表
type HotScoreOutbox struct {
	ID          uint       `gorm:"primaryKey"`
	EventID     string     `gorm:"size:64;not null;uniqueIndex"`
	VideoID     uint       `gorm:"not null"`
	Delta       float64    `gorm:"not null"`
	PublishedAt *time.Time `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
