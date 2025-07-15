package cron

/*
import (
	"context"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/services"
)

// MatchmakingTask 匹配任务
type MatchmakingTask struct {
	matchService *services.MatchQueueService
	modes        []string // 支持的模式列表
}

// NewMatchmakingTask 创建新的匹配任务
func NewMatchmakingTask() *MatchmakingTask {
	return &MatchmakingTask{
		matchService: services.NewMatchQueueService(),
		modes:        []string{"PvP", "Tournament"}, // 支持的模式
	}
}

// Run 执行匹配任务
func (t *MatchmakingTask) Run(ctx context.Context) {
	// 静默执行，只在成功匹配时输出日志
	for _, mode := range t.modes {
		err := t.matchService.ProcessMatchmaking(mode)
		if err != nil {
			log.Errorf("处理模式 %s 的匹配失败: %v", mode, err)
		}
		// 注意：成功匹配的日志会在 MatchQueueService 中输出
	}
}

// RegisterMatchmakingTask 注册匹配任务
func RegisterMatchmakingTask() {
	task := NewMatchmakingTask()

	// 每2秒执行一次匹配检查
	Register("matchmaking", task.Run, false, 2*time.Second)
}

*/
