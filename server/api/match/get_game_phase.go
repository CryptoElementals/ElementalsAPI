package match

import (
	"strings"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const GET_GAME_PHASE_LABEL = "GetGamePhase"

// GetGamePhaseRequest 请求结构体
type GetGamePhaseRequest struct {
	api.BaseRequest
}

// PvPInfo PvP对战信息
type PvPInfo struct {
	Phase   string `json:"Phase"`   // None, Queueing, Matching, InBattle
	MatchId string `json:"MatchId"` // 匹配ID
	RoomId  string `json:"RoomId"`  // 房间ID
}

// GetGamePhaseResponse 响应结构体
type GetGamePhaseResponse struct {
	api.BaseResponse
	Mode    string   `json:"Mode"`    // None, PvP
	PvPInfo *PvPInfo `json:"PvPInfo"` // PvP对战信息
}

type GetGamePhaseTask struct {
	Request  *GetGamePhaseRequest
	Response *GetGamePhaseResponse
}

// 解码请求
func NewGetGamePhaseRequest(data *map[string]interface{}) (*GetGamePhaseRequest, error) {
	req := &GetGamePhaseRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetGamePhaseResponse(sessionId string) *GetGamePhaseResponse {
	return &GetGamePhaseResponse{
		BaseResponse: api.BaseResponse{
			Action:      GET_GAME_PHASE_LABEL + "Response",
			RequestUUID: sessionId,
		},
		PvPInfo: &PvPInfo{
			Phase: "None",
		},
	}
}

func NewGetGamePhaseTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewGetGamePhaseRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetGamePhaseTask{
		Request:  req,
		Response: NewGetGamePhaseResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *GetGamePhaseTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址（从认证中间件设置的params中获取）
	_params, _ := c.Get("params")
	params, ok := _params.(*map[string]interface{})
	if !ok {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Parameter parsing failed"
		return task.Response, nil
	}

	address, ok := (*params)["Address"].(string)
	if !ok || address == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Failed to get player address"
		return task.Response, nil
	}

	// 将地址转换为小写，确保与数据库中存储的格式一致
	lowercaseAddress := strings.ToLower(address)

	// 创建匹配队列服务
	matchService := services.NewMatchQueueService()

	// 检查用户是否在匹配队列中（检查所有游戏模式）
	modes := []string{"PvP", "Tournament"} // 游戏模式列表
	inQueue := false
	var currentMode string

	for _, mode := range modes {
		players, err := matchService.GetQueue(mode)
		if err != nil {
			continue // 忽略错误，继续检查其他模式
		}

		for _, player := range players {
			if strings.ToLower(player.Address) == lowercaseAddress {
				inQueue = true
				currentMode = mode
				break
			}
		}
		if inQueue {
			break
		}
	}

	if inQueue {
		// 用户在队列中，阶段为Queueing
		task.Response.Mode = currentMode
		task.Response.PvPInfo.Phase = "Queueing"
		task.Response.BaseResponse.Message = "Player is in match queue"
		task.Response.BaseResponse.RetCode = 0
		return task.Response, nil
	}

	// 检查用户是否有匹配记录
	matches, err := db.GetMatchesByAddress(lowercaseAddress)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Failed to query match records"
		return task.Response, nil
	}

	// 查找用户当前活跃的匹配记录
	var activeMatch *dao.MatchPlayer
	for _, match := range matches {
		// 检查用户是玩家1还是玩家2，以及对应的状态
		userAddress := strings.ToLower(match.Player1Address)
		if userAddress == lowercaseAddress {
			if match.Player1Status == "matched" || match.Player1Status == "confirmed" {
				activeMatch = &match
				break
			}
		} else {
			userAddress = strings.ToLower(match.Player2Address)
			if userAddress == lowercaseAddress {
				if match.Player2Status == "matched" || match.Player2Status == "confirmed" {
					activeMatch = &match
					break
				}
			}
		}
	}

	if activeMatch != nil {
		task.Response.Mode = activeMatch.Mode
		task.Response.PvPInfo.MatchId = activeMatch.MatchID

		// 检查用户的状态
		userAddress := strings.ToLower(activeMatch.Player1Address)
		userStatus := activeMatch.Player1Status
		if userAddress != lowercaseAddress {
			userAddress = strings.ToLower(activeMatch.Player2Address)
			userStatus = activeMatch.Player2Status
		}

		if userStatus == "matched" {
			// 用户已匹配，等待确认
			task.Response.PvPInfo.Phase = "Matching"
			task.Response.BaseResponse.Message = "Player matched, waiting for confirmation"
		} else if userStatus == "confirmed" && activeMatch.RoomID != "" {
			// 双方已确认，进入战斗
			task.Response.PvPInfo.Phase = "InBattle"
			task.Response.PvPInfo.RoomId = activeMatch.RoomID
			task.Response.BaseResponse.Message = "Player has entered battle"
		} else if userStatus == "confirmed" && activeMatch.RoomID == "" {
			// 已确认但房间ID为空，可能是确认过程中
			task.Response.PvPInfo.Phase = "Matching"
			task.Response.BaseResponse.Message = "Player confirmed, waiting for opponent confirmation"
		} else {
			// 其他状态（如cancelled等）
			task.Response.PvPInfo.Phase = "None"
			task.Response.BaseResponse.Message = "Player has no active game"
		}
	} else {
		// 用户没有活跃的匹配记录
		task.Response.Mode = "None"
		task.Response.PvPInfo.Phase = "None"
		task.Response.BaseResponse.Message = "Player is not participating in any game"
	}

	task.Response.BaseResponse.RetCode = 0
	return task.Response, nil
}
