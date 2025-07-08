package services

import (
	"encoding/json"
	"fmt"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// PlayerInfo 玩家信息
type PlayerInfo struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
}

const (
	MATCH_QUEUE_PREFIX = "match:queue:" // 匹配队列前缀
)

// MatchQueueService 匹配队列服务
type MatchQueueService struct{}

func NewMatchQueueService() *MatchQueueService {
	return &MatchQueueService{}
}

// getQueueKey 根据model获取队列key
func (s *MatchQueueService) getQueueKey(model string) string {
	return MATCH_QUEUE_PREFIX + model
}

// JoinQueue 加入匹配队列，存储address和publickey
func (s *MatchQueueService) JoinQueue(model string, address string, publicKey string) error {
	conn := redis.GetGlobalPool().Get()
	defer conn.Close()

	queueKey := s.getQueueKey(model)

	// 检查是否已在队列
	players, err := s.GetQueue(model)
	if err != nil {
		log.Errorf("获取匹配队列失败: %v", err)
		return err
	}

	for _, player := range players {
		if player.Address == address {
			return fmt.Errorf("玩家已在匹配队列中")
		}
	}

	// 创建玩家信息
	playerInfo := PlayerInfo{
		Address:   address,
		PublicKey: publicKey,
	}

	// 序列化为JSON
	playerData, err := json.Marshal(playerInfo)
	if err != nil {
		log.Errorf("序列化玩家信息失败: %v", err)
		return err
	}

	// 加入队列（队尾）
	_, err = conn.Do("RPUSH", queueKey, playerData)
	if err != nil {
		log.Errorf("加入匹配队列失败: %v", err)
		return err
	}
	log.Infof("玩家 %s (model: %s) 成功加入匹配队列", address, model)
	return nil
}

// LeaveQueue 从队列移除address
func (s *MatchQueueService) LeaveQueue(model string, address string) error {
	conn := redis.GetGlobalPool().Get()
	defer conn.Close()

	queueKey := s.getQueueKey(model)

	// 获取队列中的所有玩家
	players, err := s.GetQueue(model)
	if err != nil {
		log.Errorf("获取匹配队列失败: %v", err)
		return err
	}

	// 找到要移除的玩家
	found := false
	for _, player := range players {
		if player.Address == address {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("玩家不在匹配队列中")
	}

	// 重新构建队列，排除要移除的玩家
	_, err = conn.Do("DEL", queueKey)
	if err != nil {
		log.Errorf("清空队列失败: %v", err)
		return err
	}

	// 重新添加其他玩家
	for _, player := range players {
		if player.Address != address {
			playerData, err := json.Marshal(player)
			if err != nil {
				log.Errorf("序列化玩家信息失败: %v", err)
				continue
			}
			_, err = conn.Do("RPUSH", queueKey, playerData)
			if err != nil {
				log.Errorf("重新添加玩家到队列失败: %v", err)
			}
		}
	}

	log.Infof("玩家 %s (model: %s) 已离开匹配队列", address, model)
	return nil
}

// GetQueue 获取当前队列所有玩家信息
func (s *MatchQueueService) GetQueue(model string) ([]PlayerInfo, error) {
	conn := redis.GetGlobalPool().Get()
	defer conn.Close()

	queueKey := s.getQueueKey(model)

	// 获取队列中的所有JSON数据
	playerDataList, err := redigo.Strings(conn.Do("LRANGE", queueKey, 0, -1))
	if err != nil {
		return nil, err
	}

	// 反序列化每个玩家信息
	var players []PlayerInfo
	for _, playerData := range playerDataList {
		var player PlayerInfo
		err := json.Unmarshal([]byte(playerData), &player)
		if err != nil {
			log.Errorf("反序列化玩家信息失败: %v", err)
			continue
		}
		players = append(players, player)
	}

	return players, nil
}

// GetQueueAddresses 获取当前队列所有address（兼容旧接口）
func (s *MatchQueueService) GetQueueAddresses(model string) ([]string, error) {
	players, err := s.GetQueue(model)
	if err != nil {
		return nil, err
	}

	var addresses []string
	for _, player := range players {
		addresses = append(addresses, player.Address)
	}

	return addresses, nil
}
