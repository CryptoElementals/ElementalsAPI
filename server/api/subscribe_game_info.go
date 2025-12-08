package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/events"
	"github.com/CryptoElementals/common/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(SUBSCRIBE_GAME_INFO_LABEL, NewSubscribeGameInfoTask, COOKIEAUTH)
}

// SubscribeGameInfoRequest 请求结构体
type SubscribeGameInfoRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`     // 临时地址
	Duration    int    `mapstructure:"Duration" validate:"min=1,max=86400"` // 连接持续时间（秒）
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

// SubscribeGameInfoResponse 响应结构体
type SubscribeGameInfoResponse struct {
	BaseResponse
	Message string `json:"message"`
}

type SubscribeGameInfoTask struct {
	Request  *SubscribeGameInfoRequest
	Response *SubscribeGameInfoResponse
	mu       sync.Mutex
	stopChan chan struct{}
}

// MatchedPlayerInfo 匹配成功事件中的玩家简要信息
type MatchedPlayerInfo struct {
	PlayerID  string `json:"PlayerID"`
	Name      string `json:"Name"`
	AvatarURL string `json:"AvatarURL"`
	IsMyself  bool   `json:"IsMyself"`
}

// 解码请求
func NewSubscribeGameInfoRequest(data *map[string]interface{}) (*SubscribeGameInfoRequest, error) {
	req := &SubscribeGameInfoRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	// 设置默认值为 86400秒（1天）
	if req.Duration == 0 {
		req.Duration = 86400
	}
	return req, nil
}

func NewSubscribeGameInfoResponse(sessionId string) *SubscribeGameInfoResponse {
	return &SubscribeGameInfoResponse{
		BaseResponse: BaseResponse{
			Action:      SUBSCRIBE_GAME_INFO_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewSubscribeGameInfoTask(data *map[string]interface{}) (Task, error) {
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

// Run 实现事件驱动的 SSE 流式响应
func (task *SubscribeGameInfoTask) Run(c *gin.Context) (Response, error) {
	// 解析 PlayerID（由中间件从会话中注入），前端只需要传临时地址
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		return nil, fmt.Errorf("player id is empty")
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid player id: %v", err)
	}

	temp_address := strings.ToLower(task.Request.TempAddress)

	// 组装 game_topic: PlayerID_tempaddress 格式
	game_topic := fmt.Sprintf("%d_%s", playerID, temp_address)

	// 获取全局事件管理器
	eventManager := events.GetGlobalEventManager()

	// 注册SSE客户端
	clientID := fmt.Sprintf("%s_%s", task.Request.RequestUUID, game_topic)
	eventHandler := func(msg *proto.Message) {
		// 将RoomServer事件转换为SSE事件并发送
		sseEvent := task.convertRoomServerEventToSSE(msg, task.Request.RequestUUID)
		if err := sendSSEEvent(c.Writer, c.Writer.(http.Flusher), sseEvent); err != nil {
			log.Errorf("发送SSE事件失败: %v", err)
		}
	}

	eventManager.RegisterSSEClient(clientID, eventHandler)
	defer eventManager.UnregisterSSEClient(clientID)

	// 订阅游戏主题
	if err := eventManager.SubscribeToTopic(clientID, game_topic); err != nil {
		log.Errorf("订阅主题失败: %v", err)
		errorEvent := events.Event{
			Type:        events.EventTypeError,
			Data:        map[string]interface{}{"error": fmt.Sprintf("订阅主题失败: %v", err)},
			Timestamp:   time.Now(),
			RequestUUID: task.Request.RequestUUID,
		}
		sendSSEEvent(c.Writer, c.Writer.(http.Flusher), errorEvent)
		return nil, err
	}
	defer eventManager.UnsubscribeFromTopic(clientID, game_topic)

	// 发送连接成功事件
	connectedEvent := events.Event{
		Type: events.EventTypeNotification,
		Data: map[string]interface{}{
			"Status": "connected",
		},
		Timestamp:   time.Now(),
		RequestUUID: task.Request.RequestUUID,
	}
	if err := sendSSEEvent(c.Writer, c.Writer.(http.Flusher), connectedEvent); err != nil {
		return nil, err
	}

	// 发送心跳保持连接
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	// 等待连接结束
	for {
		select {
		case <-c.Request.Context().Done():
			log.Infof("SSE connection closed by client - RequestUUID: %s", task.Request.RequestUUID)
			return task.Response, nil
		case <-time.After(time.Duration(task.Request.Duration) * time.Second):
			log.Infof("SSE connection timeout - RequestUUID: %s", task.Request.RequestUUID)
			return task.Response, nil
		case <-task.stopChan:
			log.Infof("SSE connection stopped manually - RequestUUID: %s", task.Request.RequestUUID)
			return task.Response, nil
		case <-ticker.C:
			// 发送心跳
			heartbeatEvent := events.Event{
				Type:        events.EventTypeHeartbeat,
				Data:        map[string]interface{}{},
				Timestamp:   time.Now(),
				RequestUUID: task.Request.RequestUUID,
			}
			if err := sendSSEEvent(c.Writer, c.Writer.(http.Flusher), heartbeatEvent); err != nil {
				log.Errorf("发送心跳失败: %v", err)
			}
		}
	}
}

// convertRoomServerEventToSSE 将RoomServer事件转换为SSE事件
func (task *SubscribeGameInfoTask) convertRoomServerEventToSSE(msg *proto.Message, requestUUID string) events.Event {
	// 根据事件类型进行转换
	switch msg.Event.Type {
	case proto.EventType_TYPE_MATCHED:
		gameMatched := msg.Event.GetGameMatched()
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "matched",
				"Message":   gameMatched,
				"Players":   task.buildMatchedPlayersInfo(gameMatched),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_PART_CONFIRMED:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "partConfirmed",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_GAME_CREATED:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "gameCreated",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_ROUND_READY:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "roundReady",
				"Message":   msg.Event.GetRoundReady(),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_TURN_READY:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "turnReady",
				"Message":   msg.Event.GetTurnReady(),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "commitmentsOnChain",
				"Message":   msg.Event.GetCommitmentsOnChain(),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_TURN_COMPLETE:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "turnComplete",
				"Message":   msg.Event.GetTurnCompleted(),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_GAME_PHASE_SYNC:
		gamePhase := msg.Event.GetGamePhase()
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "gamePhaseSync",
				"Message":   gamePhase,
				"Players":   task.buildGamePhasePlayersInfo(gamePhase),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_PLAYER_OFFLINE:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "playerOffline",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_CONTINUE_CANCELED:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "continueCanceled",
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_KNOWN:
		fallthrough
	default:
		// 对于未知事件类型，直接转发原始数据
		jsonData, _ := json.Marshal(msg)
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "unknown",
				"RawData":   string(jsonData),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	}
}

// buildMatchedPlayersInfo 根据 GameMatched 中的玩家列表补充用户名和头像信息
func (task *SubscribeGameInfoTask) buildMatchedPlayersInfo(gameMatched *proto.GameMatched) []MatchedPlayerInfo {
	if gameMatched == nil {
		return nil
	}

	players := gameMatched.GetPlayers()
	if len(players) == 0 {
		return nil
	}

	result := make([]MatchedPlayerInfo, 0, len(players))

	// 当前订阅者的 PlayerID 字符串（用于判断 IsMyself）
	currentPlayerID := ""
	if task != nil && task.Request != nil {
		currentPlayerID = strings.TrimSpace(task.Request.PlayerID)
	}
	for _, p := range players {
		if p == nil {
			continue
		}

		playerIDStr := strconv.FormatInt(p.GetId(), 10)
		info := MatchedPlayerInfo{
			PlayerID: playerIDStr,
		}

		userProfile, err := db.GetUserProfileByPlayerID(playerIDStr)
		if err != nil {
			log.Errorf("%s, failed to get user profile for matched player_id=%s: %v", requestUUIDFromTask(task), playerIDStr, err)
			result = append(result, info)
			continue
		}

		info.PlayerID = strconv.FormatInt(userProfile.PlayerID, 10)
		info.Name = userProfile.Name

		if userProfile.AvatarURL != "" {
			if avatarURL, err := utils.GetPresignedImageURL(userProfile.AvatarURL); err == nil {
				info.AvatarURL = avatarURL
			} else {
				log.Errorf("%s, failed to generate presigned avatar URL for matched player_id=%s: %v", requestUUIDFromTask(task), playerIDStr, err)
			}
		}

		// 标记是否为当前用户
		if currentPlayerID != "" && currentPlayerID == playerIDStr {
			info.IsMyself = true
		}

		result = append(result, info)
	}

	return result
}

// buildGamePhasePlayersInfo 根据 GamePhase 中的玩家列表补充用户名和头像信息
func (task *SubscribeGameInfoTask) buildGamePhasePlayersInfo(gamePhase *proto.GamePhase) []MatchedPlayerInfo {
	if gamePhase == nil {
		return nil
	}

	phasePlayers := gamePhase.GetPlayers()
	if len(phasePlayers) == 0 {
		return nil
	}

	result := make([]MatchedPlayerInfo, 0, len(phasePlayers))

	// 当前订阅者的 PlayerID 字符串（用于判断 IsMyself）
	currentPlayerID := ""
	if task != nil && task.Request != nil {
		currentPlayerID = strings.TrimSpace(task.Request.PlayerID)
	}

	for _, gp := range phasePlayers {
		if gp == nil || gp.GetAddress() == nil {
			continue
		}

		playerID := gp.GetAddress().GetId()
		playerIDStr := strconv.FormatInt(playerID, 10)

		info := MatchedPlayerInfo{
			PlayerID: playerIDStr,
		}

		userProfile, err := db.GetUserProfileByPlayerID(playerIDStr)
		if err != nil {
			log.Errorf("%s, failed to get user profile for game phase player_id=%s: %v", requestUUIDFromTask(task), playerIDStr, err)
			result = append(result, info)
			continue
		}

		info.PlayerID = strconv.FormatInt(userProfile.PlayerID, 10)
		info.Name = userProfile.Name

		if userProfile.AvatarURL != "" {
			if avatarURL, err := utils.GetPresignedImageURL(userProfile.AvatarURL); err == nil {
				info.AvatarURL = avatarURL
			} else {
				log.Errorf("%s, failed to generate presigned avatar URL for game phase player_id=%s: %v", requestUUIDFromTask(task), playerIDStr, err)
			}
		}

		// 标记是否为当前用户
		if currentPlayerID != "" && currentPlayerID == playerIDStr {
			info.IsMyself = true
		}

		result = append(result, info)
	}

	return result
}

// requestUUIDFromTask 辅助函数：从任务中安全获取 RequestUUID，仅用于日志
func requestUUIDFromTask(task *SubscribeGameInfoTask) string {
	if task == nil || task.Request == nil {
		return ""
	}
	return task.Request.RequestUUID
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
