package battle

import (
	"github.com/CryptoElementals/common/server/api"
)

const GET_BATTLE_INFO_LABEL = "GetBattleInfo"

// RegisterBattleApis 注册对战相关API
func RegisterBattleApis() {
	api.Register(SSE_EXAMPLE_LABEL, NewSSEExampleTask, api.NOAUTH)
	api.Register(SUBSCRIBE_GAME_INFO_LABEL, NewSubscribeGameInfoTask, api.COOKIEAUTH)
}
