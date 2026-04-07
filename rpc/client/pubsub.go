package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/stream"
)

type PubSubClient struct {
	room  stream.Stream
	lobby stream.Stream
	mu    sync.RWMutex
	subs  map[string]func()
}

// NewPubSubClient uses one Redis stream backend for both room and lobby topic keys.
func NewPubSubClient(st stream.Stream) *PubSubClient {
	return NewPubSubClientDual(st, st)
}

// NewPubSubClientDual allows separate stream handles (typically the same [stream.RedisStream] instance twice).
func NewPubSubClientDual(room, lobby stream.Stream) *PubSubClient {
	return &PubSubClient{
		room:  room,
		lobby: lobby,
		subs:  make(map[string]func()),
	}
}

func (c *PubSubClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, stop := range c.subs {
		stop()
	}
	c.subs = make(map[string]func())
	return nil
}

func (c *PubSubClient) Publish(topic string, event *pb.Event, metadata map[string]string) error {
	if c.room == nil {
		return fmt.Errorf("stream is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pub := pubsub.NewStreamPublisher(c.room)
	req := &pb.PublishRequest{Topic: topic, Event: event, Metadata: metadata}
	resp, err := pub.Publish(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}
	log.Debugw("pubsub published", "message_id", resp.MessageId, "topic", topic, "subscriber_count", resp.SubscriberCount)
	return nil
}

// Subscribe listens on TopicRoom and TopicLobby via Redis streams, filtering by Receivers.
func (c *PubSubClient) Subscribe(subscriberID string, self *pb.PlayerAddress, evtChan chan *pb.Event, errChan chan error) error {
	start := func(st stream.Stream, label string, streamTopic string) error {
		if st == nil {
			return nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		key := streamTopic + "|" + subscriberID + "|" + label

		sub := pubsub.NewStreamSubscriber(st)
		msgCh, stopReader, err := sub.Subscribe(ctx, streamTopic, pubsub.SubscribeOptions{})
		if err != nil {
			cancel()
			return fmt.Errorf("subscribe %s: %w", label, err)
		}

		var once sync.Once
		stopAll := func() {
			once.Do(func() {
				stopReader()
				cancel()
				c.mu.Lock()
				delete(c.subs, key)
				c.mu.Unlock()
			})
		}

		c.mu.Lock()
		c.subs[key] = stopAll
		c.mu.Unlock()

		log.Infow("pubsub subscribed", "topic", streamTopic, "subscriber_id", subscriberID, "stream", label)
		go func() {
			defer stopAll()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-msgCh:
					if !ok {
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

// Unsubscribe stops room and lobby readers for subscriberID.
func (c *PubSubClient) Unsubscribe(subscriberID string) error {
	pairs := [][2]string{{"room", pubsub.TopicRoom}, {"lobby", pubsub.TopicLobby}}
	for _, p := range pairs {
		label, topic := p[0], p[1]
		key := topic + "|" + subscriberID + "|" + label
		c.mu.Lock()
		if stop, ok := c.subs[key]; ok {
			stop()
		}
		c.mu.Unlock()
	}
	return nil
}

// ListTopics logs XLEN for known stream keys.
func (c *PubSubClient) ListTopics(pattern string) error {
	_ = pattern
	if c.room == nil {
		return fmt.Errorf("stream is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, name := range []string{pubsub.TopicRoom, pubsub.TopicLobby, pubsub.TopicRoomSettlement} {
		n, err := c.room.Len(ctx, name)
		if err != nil {
			log.Debugw("pubsub stream len", "topic", name, "err", err)
			continue
		}
		log.Debugw("pubsub stream len", "topic", name, "messages", n)
	}
	return nil
}

// GetSubscriberCount reports Redis stream length for the topic key.
func (c *PubSubClient) GetSubscriberCount(topic string) error {
	if c.room == nil {
		return fmt.Errorf("stream is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	n, err := c.room.Len(ctx, topic)
	if err != nil {
		log.Warnw("pubsub stream len failed", "topic", topic, "err", err)
		return err
	}
	log.Debugw("pubsub stream len", "topic", topic, "messages", n)
	return nil
}

// CheckRedisPing is a lightweight health probe for the stream backend.
func CheckRedisPing() error {
	return redis.Ping()
}
