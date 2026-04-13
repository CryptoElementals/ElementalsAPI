package api

import (
	"context"
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
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/event_v2"
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

var (
	subscribeGameInfoEventBusOnce sync.Once
	subscribeGameInfoEventBus     event_v2.EventBus
	subscribeGameInfoEventBusErr  error
)

func getSubscribeGameInfoEventBus() (event_v2.EventBus, error) {
	subscribeGameInfoEventBusOnce.Do(func() {
		eventStream := client.GetGlobalEventStream()
		if eventStream == nil {
			subscribeGameInfoEventBusErr = fmt.Errorf("event stream is nil")
			return
		}
		subscribeGameInfoEventBus = event_v2.NewEventBus(
			pubsub.NewStreamSubscriber(eventStream),
			pubsub.TopicRoom,
			pubsub.TopicLobby,
			pubsub.TopicTournamentRoster,
		)
	})
	return subscribeGameInfoEventBus, subscribeGameInfoEventBusErr
}

type TurnCompletedDTO struct {
	ConfirmationTimeout int64                        `json:"ConfirmationTimeout"`
	GameContinueTimeout int64                        `json:"GameContinueTimeout"`
	GameId              int64                        `json:"GameId"`
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

type RoundSubmittedCardDTO struct {
	Description         string `json:"Description"`
	ElementRelation     int32  `json:"ElementRelation"`
	MultiplierValue     uint32 `json:"MultiplierValue"`
	MultiplierTag       string `json:"MultiplierTag"`
	PlayerHealthBefore  uint32 `json:"PlayerHealthBefore"`
	PlayerHealthEnd     uint32 `json:"PlayerHealthEnd"`
	Salt                string `json:"Salt"`
	SubmittedCardId     uint32 `json:"SubmittedCardId"`
	SubmittedCommitment string `json:"SubmittedCommitment"`
}

type MatchedPlayerInfo struct {
	PlayerID  string `json:"PlayerID"`
	Name      string `json:"Name"`
	AvatarURL string `json:"AvatarURL"`
	IsMyself  bool   `json:"IsMyself"`
}

type GameReadyPlayerInfo struct {
	PlayerID  string `json:"PlayerID"`
	Name      string `json:"Name"`
	AvatarURL string `json:"AvatarURL"`
	IsMyself  bool   `json:"IsMyself"`
	InitialHP int32  `json:"InitialHP"`
	MaxHP     int32  `json:"MaxHP"`
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

	clientID := task.Request.RequestUUID
	self := &proto.PlayerAddress{Id: playerID, TemporaryAddress: temp_address}
	eventBus, err := getSubscribeGameInfoEventBus()
	if err != nil {
		log.Errorf("failed to initialize event_v2 bus: %v", err)
		errorEvent := events.Event{
			Type:        events.EventTypeError,
			Data:        map[string]interface{}{"error": fmt.Sprintf("failed to initialize event bus: %v", err)},
			Timestamp:   time.Now(),
			RequestUUID: task.Request.RequestUUID,
		}
		sendSSEEvent(c.Writer, c.Writer.(http.Flusher), errorEvent)
		return nil, err
	}
	subscriberID := event_v2.SubscriberID{Address: self, ClientID: clientID}
	msgCh, errCh := eventBus.RegisterSubscriber(subscriberID)
	defer eventBus.UnregisterSubscriber(subscriberID)

	// After stream subscribe, sync game phase only when lobby says player is currently in-game.
	lobbyClient := client.GetGlobalLobbyClient()
	if lobbyClient == nil {
		err := fmt.Errorf("gRPC lobby client not initialized")
		log.Errorf("failed to get lobby client: %v", err)
		errorEvent := events.Event{
			Type:        events.EventTypeError,
			Data:        map[string]interface{}{"error": err.Error()},
			Timestamp:   time.Now(),
			RequestUUID: task.Request.RequestUUID,
		}
		_ = sendSSEEvent(c.Writer, c.Writer.(http.Flusher), errorEvent)
		return nil, err
	}
	task.sendTournamentSnapshotSSE(c, self, lobbyClient, "")
	statusResp, err := lobbyClient.GetPlayerStatus(context.Background(), self)
	if err != nil {
		log.Errorf("failed to get player status from lobby: %v", err)
		errorEvent := events.Event{
			Type:        events.EventTypeError,
			Data:        map[string]interface{}{"error": fmt.Sprintf("Lobby GetPlayerStatus failed: %v", err)},
			Timestamp:   time.Now(),
			RequestUUID: task.Request.RequestUUID,
		}
		_ = sendSSEEvent(c.Writer, c.Writer.(http.Flusher), errorEvent)
		return nil, err
	}
	if statusResp != nil && statusResp.GetStatus() == proto.PlayerStatus_PLAYER_IN_GAME {
		rpcClient := client.GetGlobalRpcClient()
		if rpcClient == nil {
			err := fmt.Errorf("gRPC room client not initialized")
			log.Errorf("failed to get room rpc client: %v", err)
			errorEvent := events.Event{
				Type:        events.EventTypeError,
				Data:        map[string]interface{}{"error": err.Error()},
				Timestamp:   time.Now(),
				RequestUUID: task.Request.RequestUUID,
			}
			_ = sendSSEEvent(c.Writer, c.Writer.(http.Flusher), errorEvent)
			return nil, err
		}
		if _, err := rpcClient.SyncGamePhase(context.Background(), self); err != nil {
			log.Errorf("RoomServer SyncGamePhase failed: %v", err)
			errorEvent := events.Event{
				Type:        events.EventTypeError,
				Data:        map[string]interface{}{"error": fmt.Sprintf("RoomServer SyncGamePhase failed: %v", err)},
				Timestamp:   time.Now(),
				RequestUUID: task.Request.RequestUUID,
			}
			_ = sendSSEEvent(c.Writer, c.Writer.(http.Flusher), errorEvent)
			return nil, err
		}
	}

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
		case msg, ok := <-msgCh:
			if !ok {
				log.Infof("event stream closed - RequestUUID: %s", task.Request.RequestUUID)
				return task.Response, nil
			}
			if msg.GetTopic() == pubsub.TopicTournamentRoster && msg.GetEvent() != nil &&
				msg.GetEvent().GetType() == proto.EventType_TYPE_TOURNAMENT_ROSTER_UPDATE {
				task.sendTournamentSnapshotSSE(c, self, lobbyClient, msg.GetEvent().GetMessageId())
				continue
			}
			sseEvent := task.convertRoomServerEventToSSE(msg, task.Request.RequestUUID)
			if err := sendSSEEvent(c.Writer, c.Writer.(http.Flusher), sseEvent); err != nil {
				log.Errorf("failed to send SSE event: %v", err)
			}
		case err, ok := <-errCh:
			if !ok {
				return task.Response, nil
			}
			if err == nil {
				continue
			}
			log.Errorf("event bus subscriber error: %v", err)
			errorEvent := events.Event{
				Type:        events.EventTypeError,
				Data:        map[string]interface{}{"error": fmt.Sprintf("event bus subscriber error: %v", err)},
				Timestamp:   time.Now(),
				RequestUUID: task.Request.RequestUUID,
			}
			_ = sendSSEEvent(c.Writer, c.Writer.(http.Flusher), errorEvent)
			return nil, err
		}
	}
}

func (task *SubscribeGameInfoTask) convertRoomServerEventToSSE(msg *proto.Message, requestUUID string) events.Event {
	messageID := msg.Event.GetMessageId()
	switch msg.Event.Type {
	case proto.EventType_TYPE_MATCHED:
		gameMatched := msg.Event.GetGameMatched()
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"MessageID": messageID,
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
				"MessageID": messageID,
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
				"MessageID": messageID,
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
				"MessageID": messageID,
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
				"MessageID": messageID,
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
				"MessageID": messageID,
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
				"MessageID": messageID,
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
				"MessageID": messageID,
				"EventType": "gamePhaseSync",
				"Message":   gamePhase,
				"Players":   task.buildGamePhasePlayersInfo(gamePhase),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_NOT_MATCHABLE:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"MessageID": messageID,
				"EventType": "notMatchable",
				"Message":   msg.Event.GetNotMatchable(),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_MATCH_CANCELED:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"MessageID": messageID,
				"EventType": "matchCanceled",
				"Message":   msg.Event.GetMatchCanceled(),
			},
			Timestamp:   time.Now(),
			RequestUUID: requestUUID,
		}
	case proto.EventType_TYPE_GAME_SETTLEMENT_RESULT:
		return events.Event{
			Type: events.EventTypeStatusUpdate,
			Data: map[string]interface{}{
				"MessageID": messageID,
				"EventType": "gameSettlementResult",
				"Message":   msg.Event.GetGameSettlementResult(),
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
				"MessageID": messageID,
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
	maxHP := gameReady.GetMaxHP()
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
			PlayerID:  playerIDStr,
			InitialHP: int32(initialHP),
			MaxHP:     int32(maxHP),
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
				PlayerHealthBefore:  card.GetPlayerHealthBefore(),
				PlayerHealthEnd:     card.GetPlayerHealthEnd(),
				Salt:                base64.StdEncoding.EncodeToString(card.GetSalt()),
				SubmittedCardId:     card.GetSubmittedCardId(),
				SubmittedCommitment: base64.StdEncoding.EncodeToString(card.GetSubmittedCommitment()),
			}

			// 先缓存，后面根据 ElementRelation 与双方本回合结算血量差计算 MultiplierValue / MultiplierTag
			submittedCards[idx] = submittedCardDTO
		}

		dto.PlayerTurnInfos = append(dto.PlayerTurnInfos, TurnCompletedPlayerInfoDTO{
			IsMyself:      isMyself,
			PlayerAddress: addrDTO,
			SubmittedCard: submittedCardDTO,
		})
	}

	// 本回合各提交卡结算血量的 max−min，与房间端 HP spread 倍率一致（通常为 2 人对局）
	var mulVal uint32
	var minEnd, maxEnd uint32
	var nEnds int
	for _, sc := range submittedCards {
		if sc == nil {
			continue
		}
		e := sc.PlayerHealthEnd
		if nEnds == 0 {
			minEnd, maxEnd = e, e
		} else {
			if e < minEnd {
				minEnd = e
			}
			if e > maxEnd {
				maxEnd = e
			}
		}
		nEnds++
	}
	if nEnds >= 2 {
		mulVal = dao.MultiplierFromHPSpread(int64(maxEnd - minEnd))
	}
	if mulVal != 0 {
		for _, sc := range submittedCards {
			if sc == nil {
				continue
			}
			switch sc.ElementRelation {
			case 0, 4:
				sc.MultiplierTag = "bonus"
				sc.MultiplierValue = mulVal
			case 1:
				sc.MultiplierTag = "multiple"
				sc.MultiplierValue = mulVal
			default:
				// leave zero values
			}
		}
	}

	return dto
}

// sendTournamentSnapshotSSE pushes tournament_snapshot when the player is not still in an active bracket
// of an in-progress tournament (so we do not distract them from finishing the current event).
func (task *SubscribeGameInfoTask) sendTournamentSnapshotSSE(c *gin.Context, self *proto.PlayerAddress, lobbyClient proto.LobbyServiceClient, sourceMessageID string) {
	if lobbyClient == nil || self == nil || task == nil {
		return
	}
	busy, err := db.TournamentPlayerInActiveBracket(self.Id, self.TemporaryAddress)
	if err != nil {
		log.Warnf("TournamentPlayerInActiveBracket: %v", err)
		return
	}
	if busy {
		return
	}
	ctx := c.Request.Context()
	snapResp, snapErr := lobbyClient.GetLatestRegistrationOpenTournamentSnapshot(ctx, self)
	if snapErr != nil {
		log.Warnf("lobby GetLatestRegistrationOpenTournamentSnapshot failed (continuing): %v", snapErr)
		return
	}
	if snapResp == nil || !snapResp.GetHasTournament() {
		return
	}
	// Subscribe API only pushes not-started tournaments (registration open).
	if snapResp.GetTournamentStatus() != string(dao.TournamentStatusRegistrationOpen) {
		return
	}
	if sourceMessageID == "" {
		sourceMessageID = pubsub.BuildEventMessageID(&proto.Event{
			Type: proto.EventType_TYPE_TOURNAMENT_ROSTER_UPDATE,
			Event: &proto.Event_TournamentRosterUpdate{
				TournamentRosterUpdate: &proto.TournamentRosterUpdate{
					TournamentID: snapResp.GetTournamentID(),
				},
			},
		})
	}
	snapEvent := events.Event{
		Type: events.EventTypeStatusUpdate,
		Data: map[string]interface{}{
			"MessageID": sourceMessageID,
			"EventType": "tournamentSnapshot",
			"Message":   snapResp,
		},
		Timestamp:   time.Now(),
		RequestUUID: task.Request.RequestUUID,
	}
	_ = sendSSEEvent(c.Writer, c.Writer.(http.Flusher), snapEvent)
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
