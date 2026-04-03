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
	"github.com/CryptoElementals/common/pubsub"
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
	fmt.Printf("Published message %s to topic %s (subscribers: %d)\n",
		resp.MessageId, topic, resp.SubscriberCount)
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
		log.Infof("Subscribed to topic %s with ID %s (%s)\n", streamTopic, subscriberID, label)
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
