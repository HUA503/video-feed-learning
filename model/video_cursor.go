package model

import "time"

// VideoCursor 记录一页中最后一条视频的排序位置
type VideoCursor struct {
	CreatedAt time.Time `json:"created_at"` //保存上一页最后一条视频的发布时间
	ID        uint      `json:"id"`         //那条视频的 ID
}
