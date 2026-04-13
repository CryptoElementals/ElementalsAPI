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
	Register(JOIN_TOURNAMENT_LABEL, NewJoinTournamentTask, COOKIEAUTH)
}

type JoinTournamentRequest struct {
	BaseRequest
	TempAddress  string `mapstructure:"TempAddress" validate:"required"`
	PlayerID     string `mapstructure:"PlayerID" validate:"required"`
	TournamentID string `mapstructure:"TournamentID" validate:"required"`
}

type JoinTournamentResponse struct {
	BaseResponse
}

type JoinTournamentTask struct {
	Request  *JoinTournamentRequest
	Response *JoinTournamentResponse
}

func NewJoinTournamentRequest(data *map[string]interface{}) (*JoinTournamentRequest, error) {
	req := &JoinTournamentRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewJoinTournamentResponse(sessionID string) *JoinTournamentResponse {
	return &JoinTournamentResponse{
		BaseResponse: BaseResponse{
			Action:      JOIN_TOURNAMENT_LABEL + "Response",
			RequestUUID: sessionID,
		},
	}
}

func NewJoinTournamentTask(data *map[string]interface{}) (Task, error) {
	req, err := NewJoinTournamentRequest(data)
	if err != nil {
		return nil, err
	}
	task := &JoinTournamentTask{
		Request:  req,
		Response: NewJoinTournamentResponse(req.BaseRequest.RequestUUID),
	}
	if err := validator.New().Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *JoinTournamentTask) Run(c *gin.Context) (Response, error) {
	playerIDStr := strings.TrimSpace(task.Request.PlayerID)
	if playerIDStr == "" {
		task.Response.RetCode = 1001
		task.Response.Message = "player id is empty"
		return task.Response, nil
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		task.Response.RetCode = 1001
		task.Response.Message = "invalid player id"
		return task.Response, nil
	}

	tournamentID := strings.TrimSpace(task.Request.TournamentID)
	if tournamentID == "" {
		task.Response.RetCode = 1001
		task.Response.Message = "tournament id is empty"
		return task.Response, nil
	}

	lobbyClient := client.GetGlobalLobbyClient()
	if lobbyClient == nil {
		task.Response.RetCode = 1002
		task.Response.Message = "gRPC lobby client not initialized"
		return task.Response, nil
	}

	req := &proto.JoinTournamentRequest{
		PlayerAddress: &proto.PlayerAddress{
			Id:               playerID,
			TemporaryAddress: strings.ToLower(task.Request.TempAddress),
		},
		TournamentID: tournamentID,
	}

	if _, err := lobbyClient.JoinTournament(context.Background(), req); err != nil {
		task.Response.RetCode = 1002
		task.Response.Message = "Lobby JoinTournament failed: " + ShortGRPCError(err)
		return task.Response, nil
	}

	task.Response.RetCode = 0
	task.Response.Message = "Successfully joined tournament"
	return task.Response, nil
}
