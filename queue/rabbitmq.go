package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ConnectRabbitMQ(url string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(url) //建立 RabbitMQ 连接
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}

	return conn, nil
}

// 声明用于热门分数事件的队列
const HotScoreQueue = "video.hot.score"

func DeclareHotScoreQueue(conn *amqp.Connection) (*amqp.Channel, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("创建 RabbitMQ Channel 失败: %w", err)
	}

	//声明名为 HotScoreQueue 的队列
	_, err = channel.QueueDeclare(
		HotScoreQueue,
		true,  // durable：RabbitMQ 重启后保留队列
		false, // autoDelete：不自动删除
		false, // exclusive：不限制为当前连接独占
		false, // noWait：等待 RabbitMQ 确认声明结果
		nil,   // 暂无额外参数
	)
	if err != nil {
		channel.Close()
		return nil, fmt.Errorf("声明热度队列失败: %w", err)
	}

	return channel, nil
}
