package match

import (
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

const LEAVE_ROOM_LABEL = "LeaveRoom"

// LeaveRoomRequest 请求结构体
type LeaveRoomRequest struct {
	api.BaseRequest
	GameID      uint   `mapstructure:"GameID" validate:"required"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
}

// LeaveRoomResponse 响应结构体
type LeaveRoomResponse struct {
	api.BaseResponse
}

type LeaveRoomTask struct {
	Request  *LeaveRoomRequest
	Response *LeaveRoomResponse
}

func NewLeaveRoomRequest(data *map[string]interface{}) (*LeaveRoomRequest, error) {
	req := &LeaveRoomRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewLeaveRoomResponse(sessionId string) *LeaveRoomResponse {
	return &LeaveRoomResponse{
		BaseResponse: api.BaseResponse{
			Action:      LEAVE_ROOM_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewLeaveRoomTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewLeaveRoomRequest(data)
	if err != nil {
		return nil, err
	}
	task := &LeaveRoomTask{
		Request:  req,
		Response: NewLeaveRoomResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *LeaveRoomTask) Run(c *gin.Context) (api.Response, error) {
	// 获取玩家地址
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

	address = strings.ToLower(address)
	tempAddress := strings.ToLower(task.Request.TempAddress)

	log.Infof("LeaveRoom: %s, %s", address, tempAddress)

	return task.Response, nil

}
