package battle

import (
	"encoding/json"
	"fmt"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/log"
)

// BattleCache 对战缓存系统
type BattleCache struct {
	cache cache.Cache
}

// NewBattleCache 创建新的对战缓存系统
func NewBattleCache() *BattleCache {
	return &BattleCache{
		cache: cache.NewMemCache(),
	}
}

// GetDetailedActionsFromCache 从缓存获取详细动作
func (bc *BattleCache) GetDetailedActionsFromCache(roomID string, stage int) ([]BattleAction, bool) {
	cacheKey := fmt.Sprintf("battle_details_%s_%d", roomID, stage)
	if cached, err := bc.cache.Get(cacheKey); err == nil {
		var actions []BattleAction
		if json.Unmarshal([]byte(cached), &actions) == nil {
			log.Infof("Cache hit for %s", cacheKey)
			return actions, true
		}
	}
	log.Infof("Cache miss for %s", cacheKey)
	return nil, false
}

// CacheDetailedActions 缓存详细动作数据
func (bc *BattleCache) CacheDetailedActions(roomID string, stage int, actions []BattleAction) error {
	cacheKey := fmt.Sprintf("battle_details_%s_%d", roomID, stage)
	detailedActionsJSON, err := json.Marshal(actions)
	if err != nil {
		log.Errorf("Failed to marshal detailed actions for cache: %v", err)
		return err
	}

	err = bc.cache.Set(cacheKey, string(detailedActionsJSON), 3600) // 1小时 = 3600秒
	if err != nil {
		log.Errorf("Failed to cache detailed actions: %v", err)
		return err
	}

	log.Infof("Auto-cached detailed actions for %s", cacheKey)
	return nil
}
