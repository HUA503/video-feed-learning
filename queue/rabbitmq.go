package queue

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ConnectRabbitMQ(url string) (*amqp.Connection, error) {
	conn, err := amqp.DialConfig(url, amqp.Config{ //url：连接哪个 RabbitMQ、使用什么账号密码
		Heartbeat: 10 * time.Second, //amqp.Config：连接与恢复采用哪些设置
		Locale:    "en_US",

		//启用恢复的关键配置
		Recovery: &amqp.Recovery{
			ReconnectionConfig: &amqp.ReconnectionConfig{
				MaxRetryCount: 300,
				RetryInterval: 2 * time.Second,
			},
			TopologyRecoveryMode: amqp.TopologyRecoveryAllEnabled,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}

	return conn, nil
}

const (
	HotScoreQueue     = "video.hot.score"      // 声明用于热门分数事件的队列
	HotScoreDeadQueue = "video.hot.score.dead" //存放无效热度消息的队列
)

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

	// 声明用于存放无效热度消息的队列
	_, err = channel.QueueDeclare(
		HotScoreDeadQueue,
		true,  // durable：持久化队列
		false, // autoDelete：不自动删除
		false, // exclusive：不限制当前连接独占
		false, // noWait：等待声明结果
		nil,
	)
	if err != nil {
		channel.Close()
		return nil, fmt.Errorf("声明热度死信队列失败: %w", err)
	}
	return channel, nil
}
