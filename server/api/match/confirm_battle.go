package match

import (
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
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
		task.Response.BaseResponse.Message = "参数解析失败"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "未获取到玩家地址"
		return task.Response, nil
	}

	// 根据MatchID获取匹配记录
	matches, err := db.GetMatchesByMatchID(task.Request.MatchID)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "匹配记录不存在"
		return task.Response, nil
	}

	// 验证玩家是否是该匹配的参与者
	found := false
	var playerMatch dao.MatchPlayer
	for _, match := range matches {
		if match.Address == address {
			found = true
			playerMatch = match
			break
		}
	}

	if !found {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "您不是该匹配的参与者"
		return task.Response, nil
	}

	// 检查玩家是否已经确认过
	if playerMatch.Status == "confirmed" {
		task.Response.BaseResponse.RetCode = 1006
		task.Response.BaseResponse.Message = "您已经确认过战斗"
		return task.Response, nil
	}

	// 验证匹配状态
	if playerMatch.Status != "matched" {
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = "匹配状态不正确，无法确认战斗"
		return task.Response, nil
	}

	// 更新玩家确认状态
	err = db.UpdatePlayerStatus(task.Request.MatchID, address, "confirmed")
	if err != nil {
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "更新确认状态失败"
		return task.Response, nil
	}

	// 检查双方是否都已确认
	bothConfirmed, err := db.CheckBothPlayersConfirmed(task.Request.MatchID)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1007
		task.Response.BaseResponse.Message = "检查确认状态失败"
		return task.Response, nil
	}

	if bothConfirmed {
		// 双方都已确认，创建房间
		roomID := uuid.New().String()

		// 更新匹配记录的RoomID和状态
		err = db.UpdateMatchRoomID(task.Request.MatchID, roomID)
		if err != nil {
			task.Response.BaseResponse.RetCode = 1008
			task.Response.BaseResponse.Message = "创建房间失败"
			return task.Response, nil
		}

		// 更新整体状态为confirmed
		err = db.UpdateMatchStatus(task.Request.MatchID, "confirmed")
		if err != nil {
			task.Response.BaseResponse.RetCode = 1009
			task.Response.BaseResponse.Message = "更新匹配状态失败"
			return task.Response, nil
		}

		// 房间信息已记录在match表的RoomID字段中，无需额外的房间表

		task.Response.BaseResponse.RetCode = 0
		task.Response.BaseResponse.Message = "战斗确认成功，房间已创建"
	} else {
		// 只有一方确认
		task.Response.BaseResponse.RetCode = 0
		task.Response.BaseResponse.Message = "战斗确认成功，等待对方确认"
	}

	return task.Response, nil
}
