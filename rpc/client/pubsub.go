package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	pb "github.com/CryptoElementals/common/rpc/proto"
)

type PubSubClient struct {
	room  pb.PubSubServiceClient
	lobby pb.PubSubServiceClient
	mu    sync.RWMutex
	subs  map[string]context.CancelFunc
}

// NewPubSubClient subscribes only to room PubSub (lobby events will not be delivered).
func NewPubSubClient(conn *grpc.ClientConn) *PubSubClient {
	return NewPubSubClientDual(pb.NewPubSubServiceClient(conn), nil)
}

// NewPubSubClientDual subscribes to shared room and lobby topics ([pubsub.TopicRoom], [pubsub.TopicLobby]) and filters by Receivers.
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
	log.Debugw("pubsub published", "message_id", resp.MessageId, "topic", topic, "subscriber_count", resp.SubscriberCount)
	return nil
}

// Subscribe opens one stream on [pubsub.TopicRoom] (room gRPC) and one on [pubsub.TopicLobby] when lobby is configured.
// Events are delivered only when [pubsub.EventTargetsReceiver] accepts them for self.
func (c *PubSubClient) Subscribe(subscriberID string, self *pb.PlayerAddress, evtChan chan *pb.Event, errChan chan error) error {
	start := func(client pb.PubSubServiceClient, label string, streamTopic string) error {
		if client == nil {
			return nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		key := streamTopic + "|" + subscriberID + "|" + label
		c.mu.Lock()
		c.subs[key] = cancel
		c.mu.Unlock()
		req := &pb.SubscribeRequest{Topic: streamTopic, SubscriberId: subscriberID + "-" + label}
		stream, err := client.Subscribe(ctx, req)
		if err != nil {
			cancel()
			c.mu.Lock()
			delete(c.subs, key)
			c.mu.Unlock()
			return fmt.Errorf("subscribe %s: %w", label, err)
		}
		log.Infow("pubsub subscribed", "topic", streamTopic, "subscriber_id", subscriberID, "stream", label)
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
				ev := msg.GetEvent()
				if !pubsub.EventTargetsReceiver(ev, self) {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case evtChan <- ev:
				}
			}
		}()
		return nil
	}
	if err := start(c.room, "room", pubsub.TopicRoom); err != nil {
		return err
	}
	if c.lobby != nil {
		if err := start(c.lobby, "lobby", pubsub.TopicLobby); err != nil {
			return err
		}
	}
	return nil
}

// Unsubscribe stops room and lobby streams for subscriberID.
func (c *PubSubClient) Unsubscribe(subscriberID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	type pair struct {
		label string
		topic string
		cli   pb.PubSubServiceClient
	}
	pairs := []pair{{"room", pubsub.TopicRoom, c.room}}
	if c.lobby != nil {
		pairs = append(pairs, pair{"lobby", pubsub.TopicLobby, c.lobby})
	}
	for _, p := range pairs {
		if p.cli == nil {
			continue
		}
		key := p.topic + "|" + subscriberID + "|" + p.label
		c.mu.Lock()
		if cc, ok := c.subs[key]; ok {
			cc()
			delete(c.subs, key)
		}
		c.mu.Unlock()
		req := &pb.UnsubscribeRequest{Topic: p.topic, SubscriberId: subscriberID + "-" + p.label}
		_, _ = p.cli.Unsubscribe(ctx, req)
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
	for _, t := range resp.Topics {
		log.Debugw("pubsub topic", "name", t.Name, "subscribers", t.SubscriberCount, "messages", t.MessageCount, "last_message_time", t.LastMessageTime)
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
		log.Debugw("pubsub subscriber count", "topic", topic, "count", resp.SubscriberCount)
	} else {
		log.Warnw("pubsub subscriber count failed", "topic", topic, "error", resp.Error)
	}
	return nil
}
