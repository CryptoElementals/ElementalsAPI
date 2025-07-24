package match

import (
	"context"
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const GET_GAME_PHASE_LABEL = "GetGamePhase"

// 回合超时时间（秒）
const ROUND_TIMEOUT_SECONDS = 10
const INITIAL_HP = 3000
const INITIAL_MULTIPLER = 1

// GetGamePhaseRequest 请求结构体
type GetGamePhaseRequest struct {
	api.BaseRequest
	TempAddress string `mapstructure:"TempAddress" validate:"required"` // 临时地址
}

// PvPInfo PvP对战信息
type PvPInfo struct {
	Phase           uint32 `json:"Phase"`           // None, Queueing, Matching, InBattle: 0123
	GameID          uint32 `json:"GameID"`          // 游戏ID
	BeginAt         uint64 `json:"BeginAt"`         // 开始时间
	TimeoutDuration uint64 `json:"TimeoutDuration"` // 超时时间
}

// GetGamePhaseResponse 响应结构体
type GetGamePhaseResponse struct {
	api.BaseResponse
	Mode    uint32        `json:"Mode"`              // 0:None, 1:PvP
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
			Phase: 0,
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
	address = strings.ToLower(address)
	tempAddress := strings.ToLower(task.Request.TempAddress)

	// 通过gRPC调用RoomServer的GetPlayerInfo
	conn, err := grpc.NewClient(config.RoomServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Failed to connect to RoomServer: " + err.Error()
		return task.Response, nil
	}
	defer conn.Close()
	client := proto.NewRpcServiceClient(conn)

	playerAddr := &proto.PlayerAddress{
		WalletAddress:    address,
		TemporaryAddress: tempAddress,
	}

	gamePhase, err := client.GetGamePhase(context.Background(), playerAddr)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "RoomServer GetPlayerInfo failed: " + err.Error()
		return task.Response, nil
	}

	task.Response.PvPInfo.BeginAt = gamePhase.PvPInfo.BeginAt
	task.Response.PvPInfo.TimeoutDuration = gamePhase.PvPInfo.TimeoutDuration

	if gamePhase.PvPInfo.GameID != 0 {
		task.Response.PvPInfo.GameID = gamePhase.PvPInfo.GameID
	}
	switch gamePhase.PvPInfo.Status {
	case proto.PlayerStatus_PLAYER_IN_QUEUE:
		task.Response.Mode = uint32(gamePhase.GameType)
		task.Response.PvPInfo.Phase = 1
		task.Response.BaseResponse.Message = "Player is in match queue"
	case proto.PlayerStatus_PLAYER_MATCHED:
		task.Response.Mode = uint32(gamePhase.GameType)
		task.Response.PvPInfo.Phase = 2
		task.Response.BaseResponse.Message = "Player matched, waiting for confirmation"
	case proto.PlayerStatus_PLAYER_IN_GAME:
		task.Response.Mode = uint32(gamePhase.GameType)
		task.Response.PvPInfo.Phase = 3
		task.Response.BaseResponse.Message = "Player has entered battle"

	default:
		task.Response.Mode = 0
		task.Response.PvPInfo.Phase = 0
		task.Response.BaseResponse.Message = "Player is not participating in any game"
	}

	// 补充玩家信息
	if task.Response.PvPInfo.GameID != 0 {
		players := make([]MatchPlayer, 0)
		for _, p := range gamePhase.Players {
			userProfile, err := db.GetUserProfileByAddress(p.Address.WalletAddress)
			if err != nil {
				continue
			}
			players = append(players, MatchPlayer{
				Address:          p.Address.WalletAddress,
				IsMyself:         p.Address.WalletAddress == address,
				Confirmed:        p.IsConfirmed,
				Name:             userProfile.Name,
				AvatarURL:        userProfile.AvatarURL,
				InitialHP:        INITIAL_HP,
				InitialMultipler: INITIAL_MULTIPLER,
			})
		}
		task.Response.Players = players
	}

	task.Response.BaseResponse.RetCode = 0
	return task.Response, nil
}

type MatchPlayer struct {
	Address          string `json:"Address"`
	Name             string `json:"Name"`
	AvatarURL        string `json:"AvatarURL"`
	IsMyself         bool   `json:"IsMyself"`
	Confirmed        bool   `json:"Confirmed"`
	InitialHP        int32  `json:"InitialHP"`
	InitialMultipler int32  `json:"InitialMultipler"`
}
