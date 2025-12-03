package api

import (
	"github.com/gin-gonic/gin"
)

type AuthType uint8

const (
	_          AuthType = iota
	NOAUTH              // 无需认证
	VERIFYAUTH          // 鉴权但不使用cookie
	COOKIEAUTH          // 使用cookie认证
)

// 统一的 API Label 定义
const (
	// 登录与会话
	GET_LOGIN_CODE_LABEL    = "GetLoginCode"
	LOGIN_DILL_LABEL        = "LoginWeb3"
	REFRESH_LABEL           = "RefreshTokens"
	IS_USER_LOGGED_IN_LABEL = "IsUserLoggedIn"
	LOGOUT_LABEL            = "Logout"

	// 系统与资源
	HEALTH_CHECK_LABEL = "HealthCheck"
	GET_CARDS_LABEL    = "GetCards"
	LIST_AVATARS_LABEL = "ListAvatars"

	// 用户相关
	SET_USER_PROFILE_LABEL           = "SetUserProfile"
	GET_USER_PROFILE_LABEL           = "GetUserProfile"
	HAS_COLLECTED_DAILY_REWARD_LABEL = "HasCollectedDailyReward"
	COLLECT_DAILY_REWARD_LABEL       = "CollectDailyReward"

	// 匹配与对战
	JOIN_QUEUE_LABEL               = "JoinQueue"
	EXIT_QUEUE_LABEL               = "ExitQueue"
	CONFIRM_BATTLE_LABEL           = "ConfirmBattle"
	GET_GAME_PHASE_LABEL           = "GetGamePhase"
	REFUSE_CONTINUE_GAME_LABEL     = "RefuseContinueGame"
	CONTINUE_GAME_LABEL            = "ContinueGame"
	IS_PLAYER_IN_QUEUE_LABEL       = "IsPlayerInQueue"
	GET_BATTLE_INFO_LABEL          = "GetBattleInfo"
	SUBSCRIBE_GAME_INFO_LABEL      = "SubscribeGameInfo"
	SSE_EXAMPLE_LABEL              = "SSEExample"
	SURRENDER_LABEL                = "Surrender"
	GET_GAME_CONFIG_LABEL          = "GetGameConfig"
	SUBMIT_PLAYER_COMMITMENT_LABEL = "SubmitPlayerCommitment"
	SUBMIT_PLAYER_CARD_LABEL       = "SubmitPlayerCard"
	GET_PLAYER_STATUS_LABEL        = "GetPlayerStatus"
	GET_GAME_TIMEOUT_CONFIG_LABEL  = "GetGameTimeoutConfig"

	EXCHANGE_TOKEN_LABEL = "ExchangeToken"
)

type Task interface {
	Run(c *gin.Context) (Response, error)
}

type creator func(data *map[string]interface{}) (Task, error)

type component struct {
	creator  creator
	authType AuthType
}

var _factory = make(map[string]component)

func Register(action string, createHandler creator, authType AuthType) {
	_factory[action] = component{
		creator:  createHandler,
		authType: authType,
	}
}

func NewTask(action string, data *map[string]interface{}) (Task, error) {
	return _factory[action].creator(data)
}

func GetAllAction() []string {
	l := make([]string, 0)
	for a := range _factory {
		l = append(l, a)
	}
	return l
}

func Exist(action string) bool {
	_, ok := _factory[action]
	return ok
}

func GetActionAuthType(action string) AuthType {
	return _factory[action].authType
}
