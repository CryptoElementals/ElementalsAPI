package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/CryptoElementals/common/log"
	pb "github.com/CryptoElementals/common/rpc/proto"
)

type PubSubClient struct {
	room  pb.PubSubServiceClient
	lobby pb.PubSubServiceClient
	conn  *grpc.ClientConn
	mu    sync.RWMutex
	subs  map[string]context.CancelFunc
}

func DailGrpcEndpoint(serverAddr string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	return conn, nil
}

// NewPubSubClient subscribes only to room PubSub (lobby events will not be delivered).
func NewPubSubClient(conn *grpc.ClientConn) *PubSubClient {
	return NewPubSubClientDual(pb.NewPubSubServiceClient(conn), nil)
}

// NewPubSubClientDual subscribes to room and optionally lobby PubSub for the same player topic.
func NewPubSubClientDual(room, lobby pb.PubSubServiceClient) *PubSubClient {
	return &PubSubClient{
		room:  room,
		lobby: lobby,
		subs:  make(map[string]context.CancelFunc),
	}
}

func (c *PubSubClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cancel := range c.subs {
		cancel()
	}
	return nil
}

func (c *PubSubClient) Publish(topic string, event *pb.Event, metadata map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := &pb.PublishRequest{Topic: topic, Event: event, Metadata: metadata}
	resp, err := c.room.Publish(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to publish: %v", err)
	}
	fmt.Printf("Published message %s to topic %s (subscribers: %d)\n",
		resp.MessageId, topic, resp.SubscriberCount)
	return nil
}

func (c *PubSubClient) Subscribe(topic, subscriberID string, evtChan chan *pb.Event, errChan chan error) error {
	start := func(client pb.PubSubServiceClient, label string) error {
		if client == nil {
			return nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		key := topic + "|" + subscriberID + "|" + label
		c.mu.Lock()
		c.subs[key] = cancel
		c.mu.Unlock()
		req := &pb.SubscribeRequest{Topic: topic, SubscriberId: subscriberID + "-" + label}
		stream, err := client.Subscribe(ctx, req)
		if err != nil {
			cancel()
			c.mu.Lock()
			delete(c.subs, key)
			c.mu.Unlock()
			return fmt.Errorf("subscribe %s: %w", label, err)
		}
		log.Infof("Subscribed to topic %s with ID %s (%s)\n", topic, subscriberID, label)
		go func() {
			for {
				msg, err := stream.Recv()
				if err == io.EOF || ctx.Err() != nil {
					return
				}
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}
				select {
				case <-ctx.Done():
					return
				case evtChan <- msg.Event:
				}
			}
		}()
		return nil
	}
	if err := start(c.room, "room"); err != nil {
		return err
	}
	if c.lobby != nil {
		if err := start(c.lobby, "lobby"); err != nil {
			return err
		}
	}
	return nil
}

func (c *PubSubClient) Unsubscribe(topic, subscriberID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, label := range []string{"room", "lobby"} {
		key := topic + "|" + subscriberID + "|" + label
		c.mu.Lock()
		if cc, ok := c.subs[key]; ok {
			cc()
			delete(c.subs, key)
		}
		c.mu.Unlock()
		var client pb.PubSubServiceClient
		switch label {
		case "room":
			client = c.room
		case "lobby":
			client = c.lobby
		}
		if client == nil {
			continue
		}
		req := &pb.UnsubscribeRequest{Topic: topic, SubscriberId: subscriberID + "-" + label}
		_, _ = client.Unsubscribe(ctx, req)
	}
	return nil
}

func (c *PubSubClient) ListTopics(pattern string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := &pb.ListTopicsRequest{Pattern: pattern}
	resp, err := c.room.ListTopics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to list topics: %v", err)
	}
	fmt.Printf("Topics (%d):\n", len(resp.Topics))
	for _, topic := range resp.Topics {
		fmt.Printf("  - %s (subscribers: %d, messages: %d)\n",
			topic.Name, topic.SubscriberCount, topic.MessageCount)
		if topic.LastMessageTime > 0 {
			fmt.Printf("    Last message: %s\n",
				time.Unix(topic.LastMessageTime, 0).Format("2006-01-02 15:04:05"))
		}
	}
	return nil
}

func (c *PubSubClient) GetSubscriberCount(topic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := &pb.GetSubscriberCountRequest{Topic: topic}
	resp, err := c.room.GetSubscriberCount(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get subscriber count: %v", err)
	}
	if resp.Success {
		fmt.Printf("Topic %s has %d subscribers\n", topic, resp.SubscriberCount)
	} else {
		fmt.Printf("Failed to get subscriber count for topic %s: %s\n", topic, resp.Error)
	}
	return nil
}
