package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 订阅热度队列的函数，让 RabbitMQ 能把消息交给程序        //只能接收数据的 Go 通道
func ConsumeHotScore(channel *amqp.Channel) (<-chan amqp.Delivery, error) {
	if err := channel.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("设置消费者预取数量失败: %w", err)
	}
	//向 RabbitMQ 注册消费者
	deliveries, err := channel.Consume(
		HotScoreQueue, // 从哪个队列接收消息
		"",            // 消费者名称，由 RabbitMQ 自动生成
		false,         // autoAck：关闭自动确认
		false,         // exclusive：允许其他消费者订阅
		false,         // noLocal：保持关闭
		false,         // noWait：等待订阅结果
		nil,           // 暂无额外参数
	)
	if err != nil {
		return nil, fmt.Errorf("订阅热度队列失败: %w", err)
	}

	return deliveries, nil
}
