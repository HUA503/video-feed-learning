package model

const (
	HotFeedSourceWindow = "window_hot"
	HotFeedSourceLatest = "latest"
)

// 热门接口的查询结果
type HotFeedResult struct {
	Videos   []Video `json:"videos"`
	Degraded bool    `json:"degraded"` //是否使用了降级路径
	Source   string  `json:"source"`   //列表采用什么排序来源
}
