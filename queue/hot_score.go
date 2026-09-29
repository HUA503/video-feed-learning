package queue

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type HotScoreEvent struct {
	VideoID uint    `json:"video_id"`
	Delta   float64 `json:"delta"` //分数变化
}

func PublishHotScore(channel *amqp.Channel, event HotScoreEvent) error {
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
