package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"example.com/video-feed/queue"
	"example.com/video-feed/repository"
	amqp "github.com/rabbitmq/amqp091-go"
)

type HotScoreOutboxService struct {
	outboxRepo      *repository.HotScoreOutboxRepository
	hotScoreChannel *amqp.Channel
}

// 组织读取、发送、标记这三个步骤
func NewHotScoreOutboxService(
	outboxRepo *repository.HotScoreOutboxRepository,
	hotScoreChannel *amqp.Channel,
) *HotScoreOutboxService {
	return &HotScoreOutboxService{
		outboxRepo:      outboxRepo,
		hotScoreChannel: hotScoreChannel,
	}
}

func (s *HotScoreOutboxService) PublishPending(
	ctx context.Context,
	limit int,
) error {
	// 读取一批待发送事件
	queryCtx, queryCancel := context.WithTimeout(ctx, 5*time.Second)
	events, err := s.outboxRepo.ListPending(queryCtx, limit)
	queryCancel()

	if err != nil {
		return fmt.Errorf("查询待发送事件失败: %w", err)
	}

	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return err
		}

		// 使用数据库中保存的原事件 ID 发布消息
		err := queue.PublishHotScore(
			ctx,
			s.hotScoreChannel, //发送到 video.hot.score 队列
			queue.HotScoreEvent{
				EventID:    event.EventID,
				VideoID:    event.VideoID,
				Delta:      event.Delta,
				OccurredAt: event.CreatedAt,
			},
		)
		if err != nil {
			return err
		}

		// 获得发送确认后，标记数据库记录已发布
		markCtx, markCancel := context.WithTimeout(ctx, 5*time.Second)
		marked, err := s.outboxRepo.MarkPublished(markCtx, event.ID)
		markCancel()

		if err != nil {
			return fmt.Errorf(
				"标记事件已发布失败：event_id=%s: %w",
				event.EventID,
				err,
			)
		}

		if !marked {
			log.Printf(
				"Outbox 发布状态未更新：outbox_id=%d，event_id=%s",
				event.ID,
				event.EventID,
			)
			continue
		}

		log.Printf(
			"Outbox 事件已发布并标记：outbox_id=%d，event_id=%s",
			event.ID,
			event.EventID,
		)
	}

	return nil
}
