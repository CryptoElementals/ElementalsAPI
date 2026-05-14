package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(CONFIRM_MATCH_LABEL, NewConfirmMatchTask, COOKIEAUTH)
}

type ConfirmMatchRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
	MatchID     int64  `mapstructure:"MatchID" validate:"required"`
}

type ConfirmMatchResponse struct {
	BaseResponse
}

type ConfirmMatchTask struct {
	Request  *ConfirmMatchRequest
	Response *ConfirmMatchResponse
}

func NewConfirmMatchRequest(data *map[string]interface{}) (*ConfirmMatchRequest, error) {
	req := &ConfirmMatchRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewConfirmMatchResponse(sessionID string) *ConfirmMatchResponse {
	return &ConfirmMatchResponse{
		BaseResponse: BaseResponse{
			Action:      CONFIRM_MATCH_LABEL + "Response",
			RequestUUID: sessionID,
		},
	}
}

func NewConfirmMatchTask(data *map[string]interface{}) (Task, error) {
	req, err := NewConfirmMatchRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ConfirmMatchTask{
		Request:  req,
		Response: NewConfirmMatchResponse(req.BaseRequest.RequestUUID),
	}
	if err := validator.New().Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *ConfirmMatchTask) Run(c *gin.Context) (Response, error) {
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "player id is empty"
		return task.Response, nil
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1001
		task.Response.BaseResponse.Message = "invalid player id"
		return task.Response, nil
	}

	lobbyClient := client.GetGlobalLobbyClient()
	if lobbyClient == nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "gRPC lobby client not initialized"
		return task.Response, nil
	}

	req := &proto.ConfirmMatchRequest{
		PlayerAddress: &proto.PlayerAddress{
			Id:               playerID,
			TemporaryAddress: strings.ToLower(task.Request.TempAddress),
		},
		MatchId: task.Request.MatchID,
	}
	_, err = lobbyClient.ConfirmMatch(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Lobby ConfirmMatch failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully confirmed match"
	return task.Response, nil
}
