package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type HotScoreEvent struct {
	VideoID uint    `json:"video_id"`
	Delta   float64 `json:"delta"`    //分数变化
	EventID string  `json:"event_id"` //给每条热度消息加一个唯一的 event_id
}

// 生成一个随机的事件 ID，给每一条新的热度事件提供身份标识
func GenerateEventID() (string, error) {
	idBytes := make([]byte, 16)

	if _, err := rand.Read(idBytes); err != nil {
		return "", fmt.Errorf("生成热度事件 ID 失败: %w", err)
	}

	return hex.EncodeToString(idBytes), nil
}

func PublishHotScore(
	ctx context.Context,
	channel *amqp.Channel,
	event HotScoreEvent,
) error {
	if event.EventID == "" {
		eventID, err := GenerateEventID()
		if err != nil {
			return err
		}

		event.EventID = eventID
	}

	//将 JSON 解析
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("编码热度消息失败: %w", err)
	}

	//可重试（RabbitMQ）
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = publishHotScoreOnce(ctx, channel, event, body)
		if err == nil {
			return nil
		}

		log.Printf(
			"发送热度消息失败：event_id=%s，尝试=%d/%d，err=%v",
			event.EventID,
			attempt,
			maxAttempts,
			err,
		)

		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return fmt.Errorf(
		"热度消息发送尝试 %d 次后仍失败：event_id=%s: %w",
		maxAttempts,
		event.EventID,
		err,
	)
}

func publishHotScoreOnce(
	ctx context.Context,
	channel *amqp.Channel,
	event HotScoreEvent,
	body []byte,
) error {

	// 发布消息，并取得这条消息对应的确认对象
	//confirmation 是“这条消息的发送确认对象”
	confirmation, err := channel.PublishWithDeferredConfirm(
		"",
		HotScoreQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"发送热度消息失败：event_id=%s: %w",
			event.EventID,
			err,
		)
	}

	if confirmation == nil {
		return fmt.Errorf("RabbitMQ 发送通道未开启确认模式")
	}

	// 最多等待 5 秒，获取这条消息的发送确认
	confirmCtx, cancel := context.WithTimeout( //ctx：带有 5 秒期限的上下文
		ctx,
		5*time.Second,
	)
	defer cancel()

	acked, err := confirmation.WaitContext(confirmCtx)
	if err != nil {
		return fmt.Errorf(
			"等待热度消息确认失败：event_id=%s: %w",
			event.EventID,
			err,
		)
	}

	if !acked {
		return fmt.Errorf(
			"热度消息未获得 RabbitMQ 肯定确认：event_id=%s",
			event.EventID,
		)
	}

	log.Printf(
		"热度消息已获得发送确认：event_id=%s",
		event.EventID,
	)

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
