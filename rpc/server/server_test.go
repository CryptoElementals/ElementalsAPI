package server

import (
	"context"
	"sync"
	"testing"

	pb "github.com/CryptoElementals/common/rpc/proto"
)

func TestPubSubServer(t *testing.T) {
	server := NewPubSub()

	// 测试发布消息
	t.Run("Publish", func(t *testing.T) {
		req := &pb.PublishRequest{
			Topic: "test-topic",
			Event: &pb.Event{},
		}

		resp, err := server.Publish(context.Background(), req)
		if err != nil {
			t.Fatalf("Publish failed: %v", err)
		}

		if !resp.Success {
			t.Errorf("Expected success=true, got %v", resp.Success)
		}

		if resp.SubscriberCount != 0 {
			t.Errorf("Expected subscriber count=0, got %d", resp.SubscriberCount)
		}
	})

	// 测试获取订阅者数量
	t.Run("GetSubscriberCount", func(t *testing.T) {
		req := &pb.GetSubscriberCountRequest{
			Topic: "test-topic",
		}

		resp, err := server.GetSubscriberCount(context.Background(), req)
		if err != nil {
			t.Fatalf("GetSubscriberCount failed: %v", err)
		}

		if !resp.Success {
			t.Errorf("Expected success=true, got %v", resp.Success)
		}

		if resp.SubscriberCount != 0 {
			t.Errorf("Expected subscriber count=0, got %d", resp.SubscriberCount)
		}
	})

	// 测试列出主题
	t.Run("ListTopics", func(t *testing.T) {
		req := &pb.ListTopicsRequest{}

		resp, err := server.ListTopics(context.Background(), req)
		if err != nil {
			t.Fatalf("ListTopics failed: %v", err)
		}

		if len(resp.Topics) != 1 {
			t.Errorf("Expected 1 topic, got %d", len(resp.Topics))
		}

		if resp.Topics[0].Name != "test-topic" {
			t.Errorf("Expected topic name 'test-topic', got %s", resp.Topics[0].Name)
		}
	})

	// 测试取消订阅
	t.Run("Unsubscribe", func(t *testing.T) {
		req := &pb.UnsubscribeRequest{
			Topic:        "test-topic",
			SubscriberId: "test-subscriber",
		}

		resp, err := server.Unsubscribe(context.Background(), req)
		if err != nil {
			t.Fatalf("Unsubscribe failed: %v", err)
		}

		if !resp.Success {
			t.Errorf("Expected success=true, got %v", resp.Success)
		}
	})
}

func TestPubSubServerConcurrency(t *testing.T) {
	server := NewPubSub()

	// 并发发布消息
	t.Run("ConcurrentPublish", func(t *testing.T) {
		const numGoroutines = 10
		const messagesPerGoroutine = 10

		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					req := &pb.PublishRequest{
						Topic: "concurrent-topic",
						Event: &pb.Event{},
					}
					_, err := server.Publish(context.Background(), req)
					if err != nil {
						t.Errorf("Publish failed in goroutine %d: %v", id, err)
					}
				}
			}(i)
		}

		wg.Wait()

		// 验证消息数量
		req := &pb.ListTopicsRequest{}
		resp, err := server.ListTopics(context.Background(), req)
		if err != nil {
			t.Fatalf("ListTopics failed: %v", err)
		}

		found := false
		for _, topic := range resp.Topics {
			if topic.Name == "concurrent-topic" {
				found = true
				if topic.MessageCount != int64(numGoroutines*messagesPerGoroutine) {
					t.Errorf("Expected %d messages, got %d",
						numGoroutines*messagesPerGoroutine, topic.MessageCount)
				}
				break
			}
		}

		if !found {
			t.Error("concurrent-topic not found")
		}
	})
}

func TestPubSubServerValidation(t *testing.T) {
	server := NewPubSub()

	t.Run("PublishValidation", func(t *testing.T) {
		// 测试空主题
		req := &pb.PublishRequest{
			Topic: "",
			Event: &pb.Event{},
		}
		_, err := server.Publish(context.Background(), req)
		if err == nil {
			t.Error("Expected error for empty topic")
		}

		// 测试空消息
		req = &pb.PublishRequest{
			Topic: "test-topic",
			Event: &pb.Event{},
		}
		_, err = server.Publish(context.Background(), req)
		if err == nil {
			t.Error("Expected error for empty message")
		}
	})

	t.Run("UnsubscribeValidation", func(t *testing.T) {
		// 测试空参数
		req := &pb.UnsubscribeRequest{
			Topic:        "",
			SubscriberId: "test-subscriber",
		}
		_, err := server.Unsubscribe(context.Background(), req)
		if err == nil {
			t.Error("Expected error for empty topic")
		}

		req = &pb.UnsubscribeRequest{
			Topic:        "test-topic",
			SubscriberId: "",
		}
		_, err = server.Unsubscribe(context.Background(), req)
		if err == nil {
			t.Error("Expected error for empty subscriber_id")
		}
	})

	t.Run("GetSubscriberCountValidation", func(t *testing.T) {
		// 测试空主题
		req := &pb.GetSubscriberCountRequest{
			Topic: "",
		}
		_, err := server.GetSubscriberCount(context.Background(), req)
		if err == nil {
			t.Error("Expected error for empty topic")
		}
	})
}
