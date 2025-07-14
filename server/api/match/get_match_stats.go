package match

import (
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/services"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const GET_MATCH_STATS_LABEL = "GetMatchStats"

// GetMatchStatsRequest 请求结构体
type GetMatchStatsRequest struct {
	api.BaseRequest
	Mode string `mapstructure:"Mode" validate:"required"`
}

// GetMatchStatsResponse 响应结构体
type GetMatchStatsResponse struct {
	api.BaseResponse
	QueueCount     int `json:"queue_count"`     // 该模式在队列中匹配的人数
	WaitingConfirm int `json:"waiting_confirm"` // 该模式等待确认的人数
	InBattle       int `json:"in_battle"`       // 该模式正在对战的人数
}

type GetMatchStatsTask struct {
	Request  *GetMatchStatsRequest
	Response *GetMatchStatsResponse
}

// 解码请求
func NewGetMatchStatsRequest(data *map[string]interface{}) (*GetMatchStatsRequest, error) {
	req := &GetMatchStatsRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewGetMatchStatsResponse(sessionId string) *GetMatchStatsResponse {
	return &GetMatchStatsResponse{
		BaseResponse: api.BaseResponse{
			Action:      GET_MATCH_STATS_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewGetMatchStatsTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewGetMatchStatsRequest(data)
	if err != nil {
		return nil, err
	}
	task := &GetMatchStatsTask{
		Request:  req,
		Response: NewGetMatchStatsResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *GetMatchStatsTask) Run(c *gin.Context) (api.Response, error) {
	log.Infof("[GetMatchStats] Processing request for mode: %s", task.Request.Mode)

	// 验证游戏模式
	validModes := []string{"PvP", "Tournament"}
	modeValid := false
	for _, validMode := range validModes {
		if task.Request.Mode == validMode {
			modeValid = true
			break
		}
	}
	if !modeValid {
		log.Infof("[GetMatchStats] Invalid game mode: %s", task.Request.Mode)
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "Invalid game mode. Only PvP and Tournament are supported"
		return task.Response, nil
	}

	// 1. 获取该模式在队列中匹配的人数（从redis中获取）
	matchService := services.NewMatchQueueService()
	players, err := matchService.GetQueue(task.Request.Mode)
	if err != nil {
		log.Infof("[GetMatchStats] Error getting queue for mode %s: %v", task.Request.Mode, err)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Failed to get queue status: " + err.Error()
		return task.Response, nil
	}
	task.Response.QueueCount = len(players)

	// 2. 获取该模式等待确认的人数（从match表里roomid为空的记录条数）
	var waitingMatches []dao.Match
	err = db.Get().Where("mode = ? AND room_id = '' AND status = 'matched'", task.Request.Mode).Find(&waitingMatches).Error
	if err != nil {
		log.Infof("[GetMatchStats] Error getting waiting matches for mode %s: %v", task.Request.Mode, err)
		task.Response.WaitingConfirm = 0
	} else {
		task.Response.WaitingConfirm = len(waitingMatches)
	}

	// 3. 获取该模式正在对战的人数（从match表里status为confirmed的记录条数）
	var confirmedMatches []dao.Match
	err = db.Get().Where("mode = ? AND status = 'confirmed'", task.Request.Mode).Find(&confirmedMatches).Error
	if err != nil {
		log.Infof("[GetMatchStats] Error getting confirmed matches for mode %s: %v", task.Request.Mode, err)
		task.Response.InBattle = 0
	} else {
		task.Response.InBattle = len(confirmedMatches)
	}

	log.Infof("[GetMatchStats] Mode %s: queue=%d, waiting_confirm=%d, in_battle=%d",
		task.Request.Mode, task.Response.QueueCount, task.Response.WaitingConfirm, task.Response.InBattle)

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully retrieved match stats"

	return task.Response, nil
}
