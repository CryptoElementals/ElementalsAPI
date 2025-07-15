package match

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

const CONFIRM_BATTLE_LABEL = "ConfirmBattle"

// ConfirmBattleRequest 请求结构体
type ConfirmBattleRequest struct {
	api.BaseRequest
	MatchID string `mapstructure:"MatchId" validate:"required"`
}

// ConfirmBattleResponse 响应结构体
type ConfirmBattleResponse struct {
	api.BaseResponse
}

type ConfirmBattleTask struct {
	Request  *ConfirmBattleRequest
	Response *ConfirmBattleResponse
}

// 解码请求
func NewConfirmBattleRequest(data *map[string]interface{}) (*ConfirmBattleRequest, error) {
	req := &ConfirmBattleRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewConfirmBattleResponse(sessionId string) *ConfirmBattleResponse {
	return &ConfirmBattleResponse{
		BaseResponse: api.BaseResponse{
			Action:      CONFIRM_BATTLE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewConfirmBattleTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewConfirmBattleRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ConfirmBattleTask{
		Request:  req,
		Response: NewConfirmBattleResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *ConfirmBattleTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to parse parameters"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address"
		return task.Response, nil
	}

	// 统一将地址转为小写
	address = strings.ToLower(address)
	log.Infof("[ConfirmBattle] Processing request for address: %s, matchId: %s", address, task.Request.MatchID)

	// 根据MatchID获取匹配记录
	match, err := db.GetMatchByMatchID(task.Request.MatchID)
	if err != nil {
		log.Infof("[ConfirmBattle] Error getting match by matchId %s: %v", task.Request.MatchID, err)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Match record does not exist"
		return task.Response, nil
	}

	// 验证玩家是否是该匹配的参与者
	found := false
	var playerStatus string
	// 将数据库中的地址也转为小写进行比较
	player1Address := strings.ToLower(match.Player1Address)
	player2Address := strings.ToLower(match.Player2Address)

	if address == player1Address {
		found = true
		playerStatus = match.Player1Status
	} else if address == player2Address {
		found = true
		playerStatus = match.Player2Status
	}

	if !found {
		log.Infof("[ConfirmBattle] Address %s is not a participant in match %s", address, task.Request.MatchID)
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "You are not a participant in this match"
		return task.Response, nil
	}

	// 检查玩家是否已经确认过
	if playerStatus == "confirmed" {
		log.Infof("[ConfirmBattle] Address %s has already confirmed match %s", address, task.Request.MatchID)
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = "You have already confirmed this match"
		return task.Response, nil
	}

	// 检查匹配状态是否正确
	if playerStatus != "matched" {
		log.Infof("[ConfirmBattle] Address %s match status is %s, cannot confirm", address, playerStatus)
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "Match status is not valid for confirmation"
		return task.Response, nil
	}

	// 更新玩家状态为confirmed
	err = db.UpdatePlayerStatus(task.Request.MatchID, address, "confirmed")
	if err != nil {
		log.Infof("[ConfirmBattle] Failed to update player status for address %s in match %s: %v", address, task.Request.MatchID, err)
		task.Response.BaseResponse.RetCode = 1006
		task.Response.BaseResponse.Message = "Failed to update match status"
		return task.Response, nil
	}

	log.Infof("[ConfirmBattle] Address %s confirmed match %s", address, task.Request.MatchID)

	// 检查双方是否都已确认
	bothConfirmed, err := db.CheckBothPlayersConfirmed(task.Request.MatchID)
	if err != nil {
		log.Infof("[ConfirmBattle] Failed to check both players confirmed for match %s: %v", task.Request.MatchID, err)
		task.Response.BaseResponse.RetCode = 1007
		task.Response.BaseResponse.Message = "Failed to check match confirmation status"
		return task.Response, nil
	}

	if bothConfirmed {
		// 双方都已确认，创建房间
		roomID := uuid.New().String()
		log.Infof("[ConfirmBattle] Both players confirmed, creating room %s for match %s", roomID, task.Request.MatchID)

		// 更新匹配记录的RoomID和状态
		err = db.UpdateMatchRoomID(task.Request.MatchID, roomID)
		if err != nil {
			log.Infof("[ConfirmBattle] Failed to update match room ID for match %s: %v", task.Request.MatchID, err)
			task.Response.BaseResponse.RetCode = 1008
			task.Response.BaseResponse.Message = "Failed to create room"
			return task.Response, nil
		}

		// 发布匹配状态变化事件到Redis
		task.publishMatchStatusChange(task.Request.MatchID, "confirmed", roomID)

		// 向room表插入初始数据，后续可以增加道具功能实现新的api更新stage0阶段
		// 为玩家1创建房间记录
		room1 := &dao.Room{
			RoomID:      roomID,
			Address:     player1Address, // 统一使用小写地址
			Stage:       0,              // 初始阶段为0
			Cards:       "",             // 初始卡牌为空
			PlayerHP:    3000,           // 初始血量为3000
			Multiplier:  1.0,            // 初始倍率为1
			IsStageOver: true,           // 初始阶段0已完成
		}

		err = db.CreateRoom(room1)
		if err != nil {
			log.Infof("[ConfirmBattle] Failed to create room record for address %s in room %s: %v", room1.Address, roomID, err)
			task.Response.BaseResponse.RetCode = 1010
			task.Response.BaseResponse.Message = "Failed to create room record"
			return task.Response, nil
		}

		// 为玩家2创建房间记录
		room2 := &dao.Room{
			RoomID:      roomID,
			Address:     player2Address, // 统一使用小写地址
			Stage:       0,              // 初始阶段为0
			Cards:       "",             // 初始卡牌为空
			PlayerHP:    3000,           // 初始血量为3000
			Multiplier:  1.0,            // 初始倍率为1
			IsStageOver: true,           // 初始阶段0已完成
		}

		err = db.CreateRoom(room2)
		if err != nil {
			log.Infof("[ConfirmBattle] Failed to create room record for address %s in room %s: %v", room2.Address, roomID, err)
			task.Response.BaseResponse.RetCode = 1010
			task.Response.BaseResponse.Message = "Failed to create room record"
			return task.Response, nil
		}

		log.Infof("[ConfirmBattle] Successfully created room %s for match %s", roomID, task.Request.MatchID)
	}

	return task.Response, nil
}

// publishMatchStatusChange 发布匹配状态变化事件到Redis
func (task *ConfirmBattleTask) publishMatchStatusChange(matchID string, status string, roomID string) {
	// 创建Redis连接
	conn := redis.GetGlobalPool().Get()
	defer conn.Close()

	// 创建事件数据
	eventData := map[string]interface{}{
		"table":     "matches",
		"operation": "UPDATE",
		"record_id": matchID,
		"data": map[string]interface{}{
			"match_id": matchID,
			"status":   status,
			"room_id":  roomID,
		},
		"timestamp": time.Now().Unix(),
	}

	// 序列化事件
	jsonData, err := json.Marshal(eventData)
	if err != nil {
		log.Errorf("Failed to marshal match status change event: %v", err)
		return
	}

	// 发布到Redis频道
	_, err = conn.Do("PUBLISH", "db:matches:changes", string(jsonData))
	if err != nil {
		log.Errorf("Failed to publish match status change event: %v", err)
		return
	}

	log.Infof("Published match status change event for match %s: %s", matchID, string(jsonData))
}
