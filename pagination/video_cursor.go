package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"example.com/video-feed/model"
)

var ErrInvalidCursor = errors.New("分页游标不合法")

// 将视频分页位置编码成适合放进 URL 的字符串
func EncodeVideoCursor(cursor model.VideoCursor) (string, error) {
	body, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("编码视频游标失败: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(body), nil
}

// 将 URL 中的游标还原为视频分页位置
func DecodeVideoCursor(cursorText string) (*model.VideoCursor, error) {
	// 没有游标，表示读取第一页
	if cursorText == "" {
		return nil, nil
	}

	// 限制输入长度，避免处理不必要的超长数据
	if len(cursorText) > 512 {
		return nil, ErrInvalidCursor
	}

	// 将 Base64 URL 字符串还原为 JSON 字节数据
	body, err := base64.RawURLEncoding.DecodeString(cursorText)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	// 将 JSON 字节数据还原为结构体
	var cursor model.VideoCursor
	if err := json.Unmarshal(body, &cursor); err != nil {
		return nil, ErrInvalidCursor
	}

	// 游标必须包含有效的视频 ID 和非零发布时间
	if cursor.ID == 0 || cursor.CreatedAt.IsZero() {
		return nil, ErrInvalidCursor
	}

	return &cursor, nil
}
