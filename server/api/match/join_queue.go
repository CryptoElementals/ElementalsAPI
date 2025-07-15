package match

import (
	"context"
	"strings"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
)

const JOIN_QUEUE_LABEL = "JoinQueue"
const roomServerAddr = "127.0.0.1:50051" // TODO: 替换为实际RoomServer地址

// JoinQueueRequest 请求结构体
type JoinQueueRequest struct {
	api.BaseRequest
	Mode        string `mapstructure:"Mode" validate:"required"`
	TempAddress string `mapstructure:"TempAddress" validate:"required"`
}

// JoinQueueResponse 响应结构体
type JoinQueueResponse struct {
	api.BaseResponse
}

type JoinQueueTask struct {
	Request  *JoinQueueRequest
	Response *JoinQueueResponse
}

// 解码请求
func NewJoinQueueRequest(data *map[string]interface{}) (*JoinQueueRequest, error) {
	req := &JoinQueueRequest{}
	err := mapstructure.Decode(*data, &req)
	if err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewJoinQueueResponse(sessionId string) *JoinQueueResponse {
	return &JoinQueueResponse{
		BaseResponse: api.BaseResponse{
			Action:      JOIN_QUEUE_LABEL + "Response",
			RequestUUID: sessionId,
		},
	}
}

func NewJoinQueueTask(data *map[string]interface{}) (api.Task, error) {
	req, err := NewJoinQueueRequest(data)
	if err != nil {
		return nil, err
	}
	task := &JoinQueueTask{
		Request:  req,
		Response: NewJoinQueueResponse(req.BaseRequest.RequestUUID),
	}

	validate := validator.New()
	err = validate.Struct(task.Request)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (task *JoinQueueTask) Run(c *gin.Context) (api.Response, error) {
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
		task.Response.BaseResponse.RetCode = 1005
		task.Response.BaseResponse.Message = "Invalid game mode. Only PvP and Tournament are supported"
		return task.Response, nil
	}

	// 检查用户token数量是否足够
	userProfile, err := db.GetUserProfileByAddress(address)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Failed to get user information"
		return task.Response, nil
	}

	// 获取用户已锁定的代币总数
	totalLockedTokens, err := db.GetTotalLockedTokensByAddress(address)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1003
		task.Response.BaseResponse.Message = "Failed to get locked token information"
		return task.Response, nil
	}

	// 计算可用代币数量：用户数据库里的token减去lock_token表里该address对应记录的token总和
	availableTokens := userProfile.TokenAmount - totalLockedTokens

	// 要求用户至少有10000个可用代币才能加入匹配队列
	if availableTokens < 10000 {
		task.Response.BaseResponse.RetCode = 1004
		task.Response.BaseResponse.Message = "Insufficient available tokens, need at least 10000 tokens to join match queue"
		return task.Response, nil
	}

	// 创建锁定代币记录
	lockToken := &dao.LockToken{
		Address:     address,
		TempAddress: tempAddress,
		Token:       10000, // 默认锁定10000个代币
	}

	// 保存锁定代币记录
	err = db.CreateLockToken(lockToken)
	if err != nil {
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "Failed to create lock token record: " + err.Error()
		return task.Response, nil
	}

	// 通过gRPC调用RoomServer的JoinQueue
	conn, err := grpc.Dial(roomServerAddr, grpc.WithInsecure())
	if err != nil {
		_ = db.SoftDeleteLockToken(lockToken.ID)
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

	_, err = client.JoinQueue(context.Background(), playerAddr)
	if err != nil {
		_ = db.SoftDeleteLockToken(lockToken.ID)
		task.Response.BaseResponse.RetCode = 1002
		task.Response.BaseResponse.Message = "RoomServer JoinQueue failed: " + err.Error()
		return task.Response, nil
	}

	// 暂时不进行匹配，只返回成功
	task.Response.BaseResponse.RetCode = 0
	task.Response.BaseResponse.Message = "Successfully joined match queue"

	return task.Response, nil
}

// RegisterMatchApis 注册匹配相关API
func RegisterMatchApis() {
	api.Register(JOIN_QUEUE_LABEL, NewJoinQueueTask, api.COOKIEAUTH)
	api.Register(GET_MATCH_STATS_LABEL, NewGetMatchStatsTask, api.NOAUTH)
	api.Register(EXIT_QUEUE_LABEL, NewExitQueueTask, api.COOKIEAUTH)
	api.Register(CONFIRM_BATTLE_LABEL, NewConfirmBattleTask, api.COOKIEAUTH)
	api.Register(CANCEL_MATCH_LABEL, NewCancelMatchTask, api.COOKIEAUTH)
	api.Register(GET_GAME_PHASE_LABEL, NewGetGamePhaseTask, api.COOKIEAUTH)
	api.Register(GET_MATCH_INFO_LABEL, NewGetMatchInfoTask, api.COOKIEAUTH)
}
