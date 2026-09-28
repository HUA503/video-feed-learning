package model

import "time"

// 关注关系
type Follow struct {
	ID uint `gorm:"primaryKey" json:"id"`
	//发起关注的人
	FollowerID uint `gorm:"not null;uniqueIndex:idx_follower_followee,priority:1" json:"follower_id"`
	//被关注的人
	FolloweeID uint      `gorm:"not null;uniqueIndex:idx_follower_followee,priority:2" json:"followee_id"`
	CreatedAt  time.Time `json:"created_at"`
}
