package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/redis"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/event_v2"
	"github.com/CryptoElementals/common/stream"
)

type PubSubClient struct {
	stream   stream.Stream
	eventBus event_v2.EventBus
	mu       sync.RWMutex
	subs     map[string]func()
}

// NewPubSubClient uses one Redis stream backend for both room and lobby topic keys.
func NewPubSubClient(st stream.Stream) *PubSubClient {
	c := &PubSubClient{
		stream: st,
		subs:   make(map[string]func()),
	}
	if st != nil {
		c.eventBus = event_v2.NewEventBus(
			pubsub.NewStreamSubscriber(st),
			pubsub.TopicRoom,
			pubsub.TopicLobby,
			pubsub.TopicTournamentRoster,
		)
	}
	return c
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
	if c.stream == nil {
		return fmt.Errorf("stream is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pub := pubsub.NewStreamPublisher(c.stream)
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
	if c.eventBus == nil {
		return fmt.Errorf("event bus is nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	key := subscriberID
	busSubscriberID := event_v2.SubscriberID{
		Address:  self,
		ClientID: key,
	}
	msgCh, busErrCh := c.eventBus.RegisterSubscriber(busSubscriberID)

	cleanup := func() {
		c.eventBus.UnregisterSubscriber(busSubscriberID)
		cancel()
		c.mu.Lock()
		delete(c.subs, key)
		c.mu.Unlock()
	}
	c.mu.Lock()
	c.subs[key] = cleanup
	c.mu.Unlock()

	log.Infow("pubsub subscribed", "topics", []string{pubsub.TopicRoom, pubsub.TopicLobby}, "subscriber_id", subscriberID)
	go func() {
		defer cleanup()
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-busErrCh:
				if !ok {
					return
				}
				if err == nil {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case errChan <- err:
				}
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				if msg == nil || msg.GetEvent() == nil {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case evtChan <- msg.GetEvent():
				}
			}
		}
	}()
	return nil
}

// Unsubscribe stops room and lobby readers for subscriberID.
func (c *PubSubClient) Unsubscribe(subscriberID string) error {
	key := subscriberID
	c.mu.Lock()
	if stop, ok := c.subs[key]; ok {
		stop()
	}
	c.mu.Unlock()
	return nil
}

// ListTopics logs XLEN for known stream keys.
func (c *PubSubClient) ListTopics(pattern string) error {
	_ = pattern
	if c.stream == nil {
		return fmt.Errorf("stream is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, name := range []string{pubsub.TopicRoom, pubsub.TopicLobby, pubsub.TopicRoomSettlementPVP} {
		n, err := c.stream.Len(ctx, name)
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
	if c.stream == nil {
		return fmt.Errorf("stream is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	n, err := c.stream.Len(ctx, topic)
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
