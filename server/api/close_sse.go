package api

import (
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(CLOSE_SSE_LABEL, NewCloseSSETask, COOKIEAUTH)
}

// CloseSSERequest 关闭 SSE 连接的请求结构
type CloseSSERequest struct {
	BaseRequest
	ClientID string `mapstructure:"ClientID" validate:"required"`
	PlayerID string `mapstructure:"PlayerID" validate:"required"`
}

// CloseSSEResponse 关闭 SSE 连接的响应结构
type CloseSSEResponse struct {
	BaseResponse
}

type CloseSSETask struct {
	Request  *CloseSSERequest
	Response *CloseSSEResponse
}

// NewCloseSSERequest 从请求参数构造 CloseSSERequest
func NewCloseSSERequest(data *map[string]interface{}) (*CloseSSERequest, error) {
	req := &CloseSSERequest{}
	if err := mapstructure.Decode(*data, req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewCloseSSEResponse(sessionId string) *CloseSSEResponse {
	return &CloseSSEResponse{
		BaseResponse: BaseResponse{
			Action:      CLOSE_SSE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewCloseSSETask(data *map[string]interface{}) (Task, error) {
	req, err := NewCloseSSERequest(data)
	if err != nil {
		return nil, err
	}
	task := &CloseSSETask{
		Request:  req,
		Response: NewCloseSSEResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}

	return task, nil
}

func (task *CloseSSETask) Run(c *gin.Context) (Response, error) {
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "player id is empty"
		return task.Response, nil
	}

	clientID := strings.TrimSpace(task.Request.ClientID)
	if clientID == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "client id is empty"
		return task.Response, nil
	}

	ok := stopSSEByClientID(clientID, playerIDStr)
	if !ok {
		log.Warnf("CloseSSE: failed to stop SSE for clientId=%s, playerId=%s", clientID, playerIDStr)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "failed to close sse connection"
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "sse connection closed"
	return task.Response, nil
}
