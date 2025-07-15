package match

import (
	"context"
	"strings"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
)

const GET_GAME_PHASE_LABEL = "GetGamePhase"

// GetGamePhaseRequest 请求结构体
type GetGamePhaseRequest struct {
	api.BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
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
	Mode    string        `json:"Mode"`              // None, PvP
	PvPInfo *PvPInfo      `json:"PvPInfo"`           // PvP对战信息
	Players []MatchPlayer `json:"Players,omitempty"` // 新增，集成对战玩家信息
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
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 通过gRPC调用RoomServer的GetPlayerInfo
	conn, err := grpc.Dial(roomServerAddr, grpc.WithInsecure())
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Failed to connect to RoomServer: " + err.Error()
		return task.Response, nil
	}
	defer conn.Close()
	client := proto.NewRpcServiceClient(conn)

	playerAddr := &proto.PlayerAddress{
		WalletAddress:    lowercaseAddress,
		TemporaryAddress: tempAddress,
	}

	playerInfo, err := client.GetPlayerInfo(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "RoomServer GetPlayerInfo failed: " + err.Error()
		return task.Response, nil
	}

	switch playerInfo.Status {
	case proto.PlayerStatus_PLAYER_IN_QUEUE:
		task.Response.Mode = "PvP"
		task.Response.PvPInfo.Phase = "Queueing"
		task.Response.BaseResponse.Message = "Player is in match queue"
	case proto.PlayerStatus_PLAYER_MATCHED:
		task.Response.Mode = "PvP"
		task.Response.PvPInfo.Phase = "Matching"
		task.Response.BaseResponse.Message = "Player matched, waiting for confirmation"
	case proto.PlayerStatus_PLAYER_IN_GAME:
		task.Response.Mode = "PvP"
		task.Response.PvPInfo.Phase = "InBattle"
		task.Response.BaseResponse.Message = "Player has entered battle"
	default:
		task.Response.Mode = "None"
		task.Response.PvPInfo.Phase = "None"
		task.Response.BaseResponse.Message = "Player is not participating in any game"
	}

	// 集成getmatchinfo功能：如果MatchId非空，查找并组装玩家信息
	if task.Response.PvPInfo.MatchId != "" {
		match, err := db.GetMatchByMatchID(task.Response.PvPInfo.MatchId)
		if err == nil {
			var players []MatchPlayer
			for _, p := range match.Players {
				addr := strings.ToLower(p.WalletAddress)
				player := MatchPlayer{
					Address:   addr,
					IsMyself:  addr == lowercaseAddress,
					Confirmed: false, // 如有状态字段可补充
				}
				userProfile, err := db.GetUserProfileByAddress(addr)
				if err == nil {
					player.Name = userProfile.Name
					player.AvatarURL = userProfile.AvatarURL
				} else {
					player.Name = addr
					player.AvatarURL = ""
				}
				players = append(players, player)
			}
			task.Response.Players = players
		}
	}

	task.Response.BaseResponse.RetCode = 0
	return task.Response, nil
}

type MatchPlayer struct {
	Address   string `json:"Address"`
	Name      string `json:"Name"`
	AvatarURL string `json:"AvatarURL"`
	IsMyself  bool   `json:"IsMyself"`
	Confirmed bool   `json:"Confirmed"`
}
