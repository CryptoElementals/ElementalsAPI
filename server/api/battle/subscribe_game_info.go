package battle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/events"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const SUBSCRIBE_GAME_INFO_LABEL = "SubscribeGameInfo"

// SubscribeGameInfoRequest 请求结构体
type SubscribeGameInfoRequest struct {
	api.BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`   // 临时地址
	Duration    int    `mapstructure:"Duration" validate:"min=1,max=360"` // 连接持续时间（秒）
}

// SubscribeGameInfoResponse 响应结构体
type SubscribeGameInfoResponse struct {
	api.BaseResponse
	Message string `json:"message"`
}

type SubscribeGameInfoTask struct {
	Request  *SubscribeGameInfoRequest
	Response *SubscribeGameInfoResponse
	mu       sync.Mutex
	stopChan chan struct{}
}

// 解码请求
func NewSubscribeGameInfoRequest(data *map[string]interface{}) (*SubscribeGameInfoRequest, error) {
	req := &SubscribeGameInfoRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	// 设置默认值为 600秒（10 分钟）
	if req.Duration == 0 {
		req.Duration = 600
	}
	return req, nil
}

func NewSubscribeGameInfoResponse(sessionId string) *SubscribeGameInfoResponse {
	return &SubscribeGameInfoResponse{
		BaseResponse: api.BaseResponse{
			Action:      SUBSCRIBE_GAME_INFO_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSubscribeGameInfoTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewSubscribeGameInfoRequest(data)
	if err != nil {
		return nil, err
	}
	task := &SubscribeGameInfoTask{
		Request:  req,
		Response: NewSubscribeGameInfoResponse(req.BaseRequest.RequestUUID),
		stopChan: make(chan struct{}, 1),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Run 实现普通的 HTTP 响应
func (task *SubscribeGameInfoTask) Run(c *gin.Context) (api.Response, error) {
	task.Response.Message = fmt.Sprintf("SubscribeGameInfo Task - TempAddress: %s, Duration: %d", task.Request.TempAddress, task.Request.Duration)
	return task.Response, nil
}

// RunSSE 实现事件驱动的 SSE 流式响应
func (task *SubscribeGameInfoTask) RunSSE(ctx context.Context, c *gin.Context, writer http.ResponseWriter, flusher http.Flusher, requestUUID string) error {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		return fmt.Errorf("parameter parsing failed")
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		return fmt.Errorf("failed to get player address")
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	lowercaseAddress := strings.ToLower(address)

	// 组装 gameID: address_tempaddress 格式
	gameID := fmt.Sprintf("%s_%s", lowercaseAddress, task.Request.TempAddress)

	// 发送开始事件
	startEvent := events.Event{
		Type: events.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"status":   "started",
			"gameId":   gameID,
			"address":  lowercaseAddress,
			"duration": task.Request.Duration,
		},
		RequestUUID: requestUUID,
	}
	if err := sendSSEEvent(writer, flusher, startEvent); err != nil {
		return err
	}

	// 启动游戏事件监听器
	done := make(chan struct{})
	task.startGameEventListener(ctx, writer, flusher, requestUUID, lowercaseAddress, gameID, done)

	// 等待连接结束
	select {
	case <-ctx.Done():
		log.Infof("SSE connection closed by client - RequestUUID: %s", requestUUID)
	case <-time.After(time.Duration(task.Request.Duration) * time.Second):
		log.Infof("SSE connection timeout - RequestUUID: %s", requestUUID)
	case <-task.stopChan:
		log.Infof("SSE connection stopped manually - RequestUUID: %s", requestUUID)
	}

	// 通知监听器退出
	close(done)

	// 发送结束事件
	endEvent := events.Event{
		Type: events.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"status": "completed",
		},
		RequestUUID: requestUUID,
	}
	if err := sendSSEEvent(writer, flusher, endEvent); err != nil {
		return err
	}

	return nil
}

// startGameEventListener 通过gRPC订阅RoomServer事件并推送SSE
func (task *SubscribeGameInfoTask) startGameEventListener(ctx context.Context, writer http.ResponseWriter, flusher http.Flusher, requestUUID string, address string, gameID string, done chan struct{}) {
	go func() {
		// 连接RoomServer
		conn, err := grpc.NewClient(roomServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Errorf("连接RoomServer失败: %v", err)
			errorEvent := events.Event{
				Type:        events.EventTypeError,
				Data:        map[string]interface{}{"error": fmt.Sprintf("连接RoomServer失败: %v", err)},
				RequestUUID: requestUUID,
			}
			sendSSEEvent(writer, flusher, errorEvent)
			return
		}
		defer conn.Close()

		// 创建PubSub客户端
		client := proto.NewPubSubServiceClient(conn)

		// 订阅游戏相关主题
		topics := []string{
			fmt.Sprintf("%s_%s", address, task.Request.TempAddress), // 使用 address_tempaddress 格式作为 topic
		}

		// 为每个主题创建订阅
		for _, topic := range topics {
			go task.subscribeToTopic(ctx, client, topic, requestUUID, writer, flusher, done)
		}

		// 发送心跳保持连接
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				heartbeatEvent := events.Event{
					Type: events.EventTypeHeartbeat,
					Data: map[string]interface{}{
						"timestamp": time.Now().Unix(),
						"gameId":    gameID,
					},
					RequestUUID: requestUUID,
				}
				sendSSEEvent(writer, flusher, heartbeatEvent)
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}()
}

// subscribeToTopic 订阅特定主题
func (task *SubscribeGameInfoTask) subscribeToTopic(ctx context.Context, client proto.PubSubServiceClient, topic string, requestUUID string, writer http.ResponseWriter, flusher http.Flusher, done chan struct{}) {
	req := &proto.SubscribeRequest{
		Topic:        topic,
		SubscriberId: fmt.Sprintf("%s_%s", requestUUID, topic),
	}

	stream, err := client.Subscribe(ctx, req)
	if err != nil {
		log.Errorf("订阅主题 %s 失败: %v", topic, err)
		return
	}

	log.Infof("成功订阅主题: %s, RequestUUID: %s", topic, requestUUID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		default:
			msg, err := stream.Recv()
			if err != nil {
				log.Errorf("接收主题 %s 消息失败: %v", topic, err)
				return
			}

			// 将RoomServer事件转换为SSE事件
			sseEvent := task.convertRoomServerEventToSSE(msg, requestUUID)
			if err := sendSSEEvent(writer, flusher, sseEvent); err != nil {
				log.Errorf("发送SSE事件失败: %v", err)
				return
			}
		}
	}
}

// convertRoomServerEventToSSE 将RoomServer事件转换为SSE事件
func (task *SubscribeGameInfoTask) convertRoomServerEventToSSE(msg *proto.Message, requestUUID string) events.Event {
	// 根据事件类型进行转换
	switch msg.Event.Type {
	case proto.EventType_SYNC_INFO:
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"eventType": "sync_info",
				"gameInfo":  msg.Event.GetGameInfo(),
			},
			RequestUUID: requestUUID,
		}
	case proto.EventType_GAME_CREATED:
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"eventType": "game_created",
				"gameId":    msg.Event.GetGameCreated().GameId,
				"players":   msg.Event.GetGameCreated().Players,
			},
			RequestUUID: requestUUID,
		}
	case proto.EventType_ROUND_READY:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"eventType": "round_ready",
				"gameId":    msg.Event.GetRoundReady().GameId,
				"roundNum":  msg.Event.GetRoundReady().RoundNum,
			},
			RequestUUID: requestUUID,
		}
	case proto.EventType_COMMITMENTS_ON_CHAIN:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"eventType": "commitments_on_chain",
				"gameId":    msg.Event.GetCommitmentsOnChain().GameId,
				"roundNum":  msg.Event.GetCommitmentsOnChain().RoundNum,
			},
			RequestUUID: requestUUID,
		}
	case proto.EventType_CARDS_ON_CHAIN:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"eventType": "cards_on_chain",
				"gameId":    msg.Event.GetCardsOnChain().GameId,
				"roundNum":  msg.Event.GetCardsOnChain().RoundNum,
			},
			RequestUUID: requestUUID,
		}
	case proto.EventType_ROUND_COMPLETE:
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"eventType": "round_complete",
				"gameId":    msg.Event.GetRoundCompleted().GameId,
				"roundInfo": msg.Event.GetRoundCompleted().RoundInfo,
			},
			RequestUUID: requestUUID,
		}
	case proto.EventType_GAME_COMPLETE:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"eventType": "game_complete",
				"gameInfo":  msg.Event.GetGameInfo(),
			},
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_KNOWN:
		fallthrough
	default:
		// 对于未知事件类型，直接转发原始数据
		jsonData, _ := json.Marshal(msg)
		return events.Event{
			Type: events.EventTypeDataChange,
			Data: map[string]interface{}{
				"eventType": "unknown",
				"rawData":   string(jsonData),
			},
			RequestUUID: requestUUID,
		}
	}
}
