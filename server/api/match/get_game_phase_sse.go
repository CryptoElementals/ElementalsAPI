package match

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/events"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
)

const GET_GAME_PHASE_SSE_LABEL = "GetGamePhaseSSE"

// GetGamePhaseSSERequest 请求结构体
type GetGamePhaseSSERequest struct {
	api.BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	Duration    int    `mapstructure:"Duration" validate:"required,min=1,max=3600"`
}

// GetGamePhaseSSEResponse 响应结构体
type GetGamePhaseSSEResponse struct {
	api.BaseResponse
	Message string `json:"message"`
}

type GetGamePhaseSSETask struct {
	Request  *GetGamePhaseSSERequest
	Response *GetGamePhaseSSEResponse
	mu       sync.Mutex
	stopChan chan struct{}
}

// 解码请求
func NewGetGamePhaseSSERequest(data *map[string]interface{}) (*GetGamePhaseSSERequest, error) {
	req := &GetGamePhaseSSERequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	// 设置默认值为 600 秒（10 分钟）
	if req.Duration == 0 {
		req.Duration = 600
	}
	return req, nil
}

func NewGetGamePhaseSSEResponse(sessionId string) *GetGamePhaseSSEResponse {
	return &GetGamePhaseSSEResponse{
		BaseResponse: api.BaseResponse{
			Action:      GET_GAME_PHASE_SSE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetGamePhaseSSETask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewGetGamePhaseSSERequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetGamePhaseSSETask{
		Request:  req,
		Response: NewGetGamePhaseSSEResponse(req.BaseRequest.RequestUUID),
		stopChan: make(chan struct{}),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Run 实现普通的 HTTP 响应
func (task *GetGamePhaseSSETask) Run(c *gin.Context) (api.Response, error) {
	task.Response.Message = fmt.Sprintf("GetGamePhase SSE Task - Duration: %d", task.Request.Duration)
	return task.Response, nil
}

// RunSSE 实现事件驱动的 SSE 流式响应
func (task *GetGamePhaseSSETask) RunSSE(ctx context.Context, c *gin.Context, writer http.ResponseWriter, flusher http.Flusher, requestUUID string) error {
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
	lowercaseTempAddress := strings.ToLower(task.Request.TempAddress)

	// 发送开始事件
	startEvent := events.Event{
		Type: events.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"status":      "started",
			"address":     lowercaseAddress,
			"tempAddress": lowercaseTempAddress,
			"duration":    task.Request.Duration,
		},
		RequestUUID: requestUUID,
	}
	if err := sendSSEEvent(writer, flusher, startEvent); err != nil {
		return err
	}

	// 启动游戏阶段监听器
	done := make(chan struct{})
	task.startGamePhaseListener(ctx, writer, flusher, requestUUID, lowercaseAddress, lowercaseTempAddress, done)

	// 等待连接结束
	select {
	case <-ctx.Done():
	case <-time.After(time.Duration(task.Request.Duration) * time.Second):
	case <-task.stopChan:
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

// startGamePhaseListener 通过gRPC订阅RoomServer事件并推送SSE
func (task *GetGamePhaseSSETask) startGamePhaseListener(ctx context.Context, writer http.ResponseWriter, flusher http.Flusher, requestUUID string, address string, tempAddress string, done chan struct{}) {
	go func() {
		const roomServerAddr = "127.0.0.1:50051" // TODO: 替换为实际RoomServer地址
		conn, err := grpc.Dial(roomServerAddr, grpc.WithInsecure())
		if err != nil {
			fmt.Printf("连接RoomServer失败: %v\n", err)
			return
		}
		defer conn.Close()
		client := proto.NewPubSubServiceClient(conn)

		topic := fmt.Sprintf("player:%s:%s", address, tempAddress)
		req := &proto.SubscribeRequest{
			Topic:        topic,
			SubscriberId: requestUUID,
		}

		stream, err := client.Subscribe(ctx, req)
		if err != nil {
			fmt.Printf("订阅RoomServer失败: %v\n", err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			default:
				msg, err := stream.Recv()
				if err != nil {
					fmt.Printf("RoomServer事件流断开: %v\n", err)
					return
				}
				jsonData, _ := json.Marshal(msg)
				writer.Write([]byte(fmt.Sprintf("data: %s\n\n", jsonData)))
				flusher.Flush()
			}
		}
	}()
}

// sendSSEEvent 发送 SSE 事件
func sendSSEEvent(writer http.ResponseWriter, flusher http.Flusher, event events.Event) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}
	// SSE 格式：data: {json}\n\n
	eventStr := fmt.Sprintf("data: %s\n\n", string(jsonData))
	_, err = writer.Write([]byte(eventStr))
	if err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
