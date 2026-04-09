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
	Register(CANCEL_MATCH_LABEL, NewCancelMatchTask, COOKIEAUTH)
}

type CancelMatchRequest struct {
	BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
	PlayerID    string `mapstructure:"PlayerID" validate:"required"`
	MatchID     int64  `mapstructure:"MatchID" validate:"required"`
}

type CancelMatchResponse struct {
	BaseResponse
}

type CancelMatchTask struct {
	Request  *CancelMatchRequest
	Response *CancelMatchResponse
}

func NewCancelMatchRequest(data *map[string]interface{}) (*CancelMatchRequest, error) {
	req := &CancelMatchRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewCancelMatchResponse(sessionID string) *CancelMatchResponse {
	return &CancelMatchResponse{
		BaseResponse: BaseResponse{
			Action:      CANCEL_MATCH_LABEL + "Response",
			RequestUUID: sessionID,
		},
	}
}

func NewCancelMatchTask(data *map[string]interface{}) (Task, error) {
	req, err := NewCancelMatchRequest(data)
	if err != nil {
		return nil, err
	}
	task := &CancelMatchTask{
		Request:  req,
		Response: NewCancelMatchResponse(req.BaseRequest.RequestUUID),
	}
	if err := validator.New().Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *CancelMatchTask) Run(c *gin.Context) (Response, error) {
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

	req := &proto.CancelMatchRequest{
		PlayerAddress: &proto.PlayerAddress{
			Id:               playerID,
			TemporaryAddress: strings.ToLower(task.Request.TempAddress),
		},
		MatchId: task.Request.MatchID,
	}
	_, err = lobbyClient.CancelMatch(context.Background(), req)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Lobby CancelMatch failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully canceled match"
	return task.Response, nil
}

