package api

import (
	"encoding/base64"
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

type SubscribeGameInfoRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	Duration    int    `mapstructure:"Duration" validate:"min=1,max=86400"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
}

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

// 全局管理当前活跃的 SubscribeGameInfo SSE 任务：clientID -> *SubscribeGameInfoTask
var sseTasks sync.Map

func registerSSETask(clientID string, task *SubscribeGameInfoTask) {
	sseTasks.Store(clientID, task)
}

func unregisterSSETask(clientID string) {
	sseTasks.Delete(clientID)
}

// stopSSEByClientID 尝试根据 clientID 主动关闭某个 SSE 连接。
// 只有当该连接的 PlayerID 与 ownerPlayerID 一致时才会真正关闭，返回 true。
func stopSSEByClientID(clientID string, ownerPlayerID string) bool {
	v, ok := sseTasks.Load(clientID)
	if !ok {
		return false
	}
	task, ok := v.(*SubscribeGameInfoTask)
	if !ok || task == nil || task.Request == nil {
		return false
	}

	if strings.TrimSpace(task.Request.PlayerID) != strings.TrimSpace(ownerPlayerID) {
		return false
	}

	task.Stop()
	return true
}

// Stop 主动停止当前 SSE 任务（通过 stopChan 通知 Run 中的 select）
func (task *SubscribeGameInfoTask) Stop() {
	task.mu.Lock()
	defer task.mu.Unlock()

	select {
	case task.stopChan <- struct{}{}:
	default:
		// 已发送过停止信号或通道已满，避免阻塞/重复发送
	}
}

type TurnCompletedDTO struct {
	ConfirmationTimeout int64                        `json:"ConfirmationTimeout"`
	GameContinueTimeout int64                        `json:"GameContinueTimeout"`
	GameId              uint32                       `json:"GameId"`
	RoundNum            uint32                       `json:"RoundNum"`
	TurnNum             uint32                       `json:"TurnNum"`
	IsRoundComplete     bool                         `json:"IsRoundComplete"`
	IsGameComplete      bool                         `json:"IsGameComplete"`
	GameResult          *proto.GameResult            `json:"GameResult,omitempty"`
	PlayerTurnInfos     []TurnCompletedPlayerInfoDTO `json:"PlayerTurnInfos"`
}

type TurnCompletedPlayerInfoDTO struct {
	IsMyself      bool                   `json:"IsMyself"`
	PlayerAddress PlayerAddressDTO       `json:"PlayerAddress"`
	SubmittedCard *RoundSubmittedCardDTO `json:"SubmittedCard,omitempty"`
}

type PlayerAddressDTO struct {
	ID               string `json:"id"`
	TemporaryAddress string `json:"temporaryAddress"`
}

type BattleEffectDTO struct {
	Description            string `json:"Description"`
	TargetPlayerId         string `json:"TargetPlayerId"`
	TargetTemporaryAddress string `json:"TargetTemporaryAddress"`
	Type                   string `json:"Type"`
	Value                  int32  `json:"Value"`
}

type RoundSubmittedCardDTO struct {
	Description         string            `json:"Description"`
	Effects             []BattleEffectDTO `json:"Effects"`
	ElementRelation     int32             `json:"ElementRelation"`
	MultiplierAfter     uint32            `json:"MultiplierAfter"`
	MultiplierBefore    uint32            `json:"MultiplierBefore"`
	MultiplierValue     uint32            `json:"MultiplierValue"`
	MultiplierTag       string            `json:"MultiplierTag"`
	PlayerHealthBefore  uint32            `json:"PlayerHealthBefore"`
	PlayerHealthEnd     uint32            `json:"PlayerHealthEnd"`
	Salt                string            `json:"Salt"`
	SubmittedCardId     uint32            `json:"SubmittedCardId"`
	SubmittedCommitment string            `json:"SubmittedCommitment"`
}

type MatchedPlayerInfo struct {
	PlayerID  string `json:"PlayerID"`
	Name      string `json:"Name"`
	AvatarURL string `json:"AvatarURL"`
	IsMyself  bool   `json:"IsMyself"`
}

type GameReadyPlayerInfo struct {
	PlayerID          string `json:"PlayerID"`
	Name              string `json:"Name"`
	AvatarURL         string `json:"AvatarURL"`
	IsMyself          bool   `json:"IsMyself"`
	InitialHP         int32  `json:"InitialHP"`
	InitialMultiplier int32  `json:"InitialMultiplier"`
}

func NewSubscribeGameInfoRequest(data *map[string]interface{}) (*SubscribeGameInfoRequest, error) {
	req := &SubscribeGameInfoRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
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

func (task *SubscribeGameInfoTask) Run(c *gin.Context) (Response, error) {
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		return nil, fmt.Errorf("player id is empty")
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid player id: %v", err)
	}

	temp_address := strings.ToLower(task.Request.TempAddress)

	game_topic := fmt.Sprintf("%d_%s", playerID, temp_address)

	eventManager := events.GetGlobalEventManager()

	clientID := fmt.Sprintf("%s_%s", task.Request.RequestUUID, game_topic)

	// 将当前 SSE 任务注册到全局表，便于后续通过 clientID 主动关闭
	registerSSETask(clientID, task)
	defer unregisterSSETask(clientID)

	eventHandler := func(msg *proto.Message) {
		sseEvent := task.convertRoomServerEventToSSE(msg, task.Request.RequestUUID)
		if err := sendSSEEvent(c.Writer, c.Writer.(http.Flusher), sseEvent); err != nil {
			log.Errorf("failed to send SSE event: %v", err)
		}
	}

	eventManager.RegisterSSEClient(clientID, eventHandler)
	defer eventManager.UnregisterSSEClient(clientID)

	// 订阅游戏主题
	if err := eventManager.SubscribeToTopic(clientID, game_topic); err != nil {
		log.Errorf("failed to subscribe to topic: %v", err)
		errorEvent := events.Event{
			Type:        events.EventTypeError,
			Data:        map[string]interface{}{"error": fmt.Sprintf("failed to subscribe to topic: %v", err)},
			Timestamp:   time.Now(),
			RequestUUID: task.Request.RequestUUID,
		}
		sendSSEEvent(c.Writer, c.Writer.(http.Flusher), errorEvent)
		return nil, err
	}
	defer eventManager.UnsubscribeFromTopic(clientID, game_topic)

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
			heartbeatEvent := events.Event{
				Type:        events.EventTypeHeartbeat,
				Data:        map[string]interface{}{},
				Timestamp:   time.Now(),
				RequestUUID: task.Request.RequestUUID,
			}
			if err := sendSSEEvent(c.Writer, c.Writer.(http.Flusher), heartbeatEvent); err != nil {
				log.Errorf("failed to send heartbeat: %v", err)
			}
		}
	}
}

func (task *SubscribeGameInfoTask) convertRoomServerEventToSSE(msg *proto.Message, requestUUID string) events.Event {
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
		gameReady := msg.Event.GetGameReady()
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "gameCreated",
				"Message":   gameReady,
				"Players":   task.buildGameReadyPlayersInfo(gameReady),
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
		turnCompleted := msg.Event.GetTurnCompleted()
		turnCompletedDTO := buildTurnCompletedDTO(task, turnCompleted)
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"EventType": "turnComplete",
				"Message":   turnCompletedDTO,
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

func (task *SubscribeGameInfoTask) buildMatchedPlayersInfo(gameMatched *proto.GameMatched) []MatchedPlayerInfo {
	if gameMatched == nil {
		return nil
	}

	players := gameMatched.GetPlayers()
	if len(players) == 0 {
		return nil
	}

	result := make([]MatchedPlayerInfo, 0, len(players))

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

		if currentPlayerID != "" && currentPlayerID == playerIDStr {
			info.IsMyself = true
		}

		result = append(result, info)
	}

	return result
}

func (task *SubscribeGameInfoTask) buildGameReadyPlayersInfo(gameReady *proto.GameReady) []GameReadyPlayerInfo {
	if gameReady == nil {
		return nil
	}

	players := gameReady.GetPlayers()
	initialHP := gameReady.GetInitialHP()
	initialMultiplier := gameReady.GetInitialMultiplier()
	if len(players) == 0 {
		return nil
	}

	result := make([]GameReadyPlayerInfo, 0, len(players))

	currentPlayerID := ""
	if task != nil && task.Request != nil {
		currentPlayerID = strings.TrimSpace(task.Request.PlayerID)
	}
	for _, p := range players {
		if p == nil {
			continue
		}

		playerIDStr := strconv.FormatInt(p.GetId(), 10)
		info := GameReadyPlayerInfo{
			PlayerID:          playerIDStr,
			InitialHP:         int32(initialHP),
			InitialMultiplier: int32(initialMultiplier),
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

		if currentPlayerID != "" && currentPlayerID == playerIDStr {
			info.IsMyself = true
		}

		result = append(result, info)
	}

	return result

}

func (task *SubscribeGameInfoTask) buildGamePhasePlayersInfo(gamePhase *proto.GamePhase) []MatchedPlayerInfo {
	if gamePhase == nil {
		return nil
	}

	phasePlayers := gamePhase.GetPlayers()
	if len(phasePlayers) == 0 {
		return nil
	}

	result := make([]MatchedPlayerInfo, 0, len(phasePlayers))

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

		if currentPlayerID != "" && currentPlayerID == playerIDStr {
			info.IsMyself = true
		}

		result = append(result, info)
	}

	return result
}

func requestUUIDFromTask(task *SubscribeGameInfoTask) string {
	if task == nil || task.Request == nil {
		return ""
	}
	return task.Request.RequestUUID
}

func buildTurnCompletedDTO(task *SubscribeGameInfoTask, tc *proto.TurnCompleted) *TurnCompletedDTO {
	if tc == nil {
		return &TurnCompletedDTO{}
	}

	dto := &TurnCompletedDTO{
		ConfirmationTimeout: tc.GetConfirmationTimeout(),
		GameContinueTimeout: tc.GetGameContinueTimeout(),
		GameId:              tc.GetGameId(),
		RoundNum:            tc.GetRoundNum(),
		TurnNum:             tc.GetTurnNum(),
		IsRoundComplete:     tc.GetIsRoundComplete(),
		IsGameComplete:      tc.GetIsGameComplete(),
	}

	// 只有在游戏结束时才挂上 GameResult，其它情况下完全不下发该字段
	if dto.IsGameComplete && tc.GetGameResult() != nil {
		dto.GameResult = tc.GetGameResult()
	}

	infos := tc.GetPlayerTurnInfos()
	if len(infos) == 0 {
		return dto
	}

	// 订阅者自身的 PlayerID 与 TempAddress，用于标记 IsMyself
	currentPlayerID := int64(0)
	if task != nil && task.Request != nil {
		if id, err := strconv.ParseInt(strings.TrimSpace(task.Request.PlayerID), 10, 64); err == nil {
			currentPlayerID = id
		}
	}
	currentTempAddr := ""
	if task != nil && task.Request != nil {
		currentTempAddr = strings.ToLower(task.Request.TempAddress)
	}

	dto.PlayerTurnInfos = make([]TurnCompletedPlayerInfoDTO, 0, len(infos))
	submittedCards := make([]*RoundSubmittedCardDTO, len(infos))

	for idx, info := range infos {
		if info == nil {
			continue
		}

		playerAddr := info.GetPlayerAddress()
		addrDTO := PlayerAddressDTO{}
		isMyself := false

		if playerAddr != nil {
			addrDTO.ID = strconv.FormatInt(playerAddr.GetId(), 10)
			addrDTO.TemporaryAddress = playerAddr.GetTemporaryAddress()

			addrID := playerAddr.GetId()
			addrTemp := strings.ToLower(playerAddr.GetTemporaryAddress())
			if currentPlayerID != 0 && addrID == currentPlayerID &&
				currentTempAddr != "" && addrTemp == currentTempAddr {
				isMyself = true
			}
		}

		var submittedCardDTO *RoundSubmittedCardDTO
		if card := info.GetSubmittedCard(); card != nil {
			submittedCardDTO = &RoundSubmittedCardDTO{
				Description: card.GetDescription(),
				// 直接使用枚举的底层数值，前端拿到的是数字
				ElementRelation:     int32(card.GetElementRelation()),
				MultiplierAfter:     card.GetMultiplierAfter(),
				MultiplierBefore:    card.GetMultiplierBefore(),
				PlayerHealthBefore:  card.GetPlayerHealthBefore(),
				PlayerHealthEnd:     card.GetPlayerHealthEnd(),
				Salt:                base64.StdEncoding.EncodeToString(card.GetSalt()),
				SubmittedCardId:     card.GetSubmittedCardId(),
				SubmittedCommitment: base64.StdEncoding.EncodeToString(card.GetSubmittedCommitment()),
			}

			// 先缓存，后面统一根据 ElementRelation 和双方 MultiplierAfter 计算 Multipliervalue / multiprefix
			submittedCards[idx] = submittedCardDTO

			effects := card.GetEffects()
			if len(effects) > 0 {
				submittedCardDTO.Effects = make([]BattleEffectDTO, 0, len(effects))
				for _, ef := range effects {
					if ef == nil {
						continue
					}
					submittedCardDTO.Effects = append(submittedCardDTO.Effects, BattleEffectDTO{
						Description:            ef.GetDescription(),
						TargetPlayerId:         strconv.FormatInt(ef.GetTargetPlayerId(), 10),
						TargetTemporaryAddress: ef.GetTargetTemporaryAddress(),
						Type:                   ef.GetType().String(),
						Value:                  ef.GetValue(),
					})
				}
			} else {
				submittedCardDTO.Effects = []BattleEffectDTO{}
			}
		}

		dto.PlayerTurnInfos = append(dto.PlayerTurnInfos, TurnCompletedPlayerInfoDTO{
			IsMyself:      isMyself,
			PlayerAddress: addrDTO,
			SubmittedCard: submittedCardDTO,
		})
	}

	// 第二轮遍历，根据 ElementRelation 计算 MultiplierValue 和 MultiplierTag
	for i, sc := range submittedCards {
		if sc == nil {
			continue
		}

		switch sc.ElementRelation {
		case 0, 4:
			// bonus：取“对方”的 MultiplierAfter
			var opponentMultiplier uint32
			for j, other := range submittedCards {
				if j == i || other == nil {
					continue
				}
				opponentMultiplier = other.MultiplierAfter
				break
			}
			if opponentMultiplier != 0 {
				sc.MultiplierTag = "bonus"
				sc.MultiplierValue = opponentMultiplier
			}
		case 1:
			// multiple：取自己的 MultiplierAfter
			sc.MultiplierTag = "multiple"
			sc.MultiplierValue = sc.MultiplierAfter
		default:
			// 其它情况不赋值，保持默认零值/空字符串
		}
	}

	return dto
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
