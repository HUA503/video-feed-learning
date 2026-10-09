package queue

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type HotScoreEvent struct {
	VideoID uint    `json:"video_id"`
	Delta   float64 `json:"delta"`    //分数变化
	EventID string  `json:"event_id"` //给每条热度消息加一个唯一的 event_id
}

func PublishHotScore(channel *amqp.Channel, event HotScoreEvent) error {
	if event.EventID == "" {
		idBytes := make([]byte, 16)

		if _, err := rand.Read(idBytes); err != nil {
			return fmt.Errorf("生成热度事件 ID 失败: %w", err)
		}

		event.EventID = hex.EncodeToString(idBytes)
	}

	//将 JSON 解析
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("编码热度消息失败: %w", err)
	}

	//发布消息
	err = channel.Publish(
		"",            // 使用 RabbitMQ 默认交换机
		HotScoreQueue, // 路由到热度队列
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("发送热度消息失败: %w", err)
	}

	return nil
}

// 消息解析函数，让后台程序能读懂队列中的 JSON
func DecodeHotScore(body []byte) (HotScoreEvent, error) {
	var event HotScoreEvent
	//把 JSON 解析进结构体
	if err := json.Unmarshal(body, &event); err != nil {
		return HotScoreEvent{}, fmt.Errorf("解析热度消息失败: %w", err)
	}

	if event.VideoID == 0 {
		return HotScoreEvent{}, fmt.Errorf("热度消息的视频 ID 必须大于 0")
	}

	if event.Delta == 0 {
		return HotScoreEvent{}, fmt.Errorf("热度消息的分数变化不能为 0")
	}

	if event.EventID == "" {
		return HotScoreEvent{}, fmt.Errorf("热度消息的事件 ID 不能为空")
	}
	return event, nil
}
