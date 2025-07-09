package services

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/redis"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

// PlayerInfo 玩家信息
type PlayerInfo struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
}

const (
	MATCH_QUEUE_PREFIX = "match:queue:" // 匹配队列前缀
	MATCH_THRESHOLD    = 2              // 匹配阈值（2人）
)

// MatchQueueService 匹配队列服务
type MatchQueueService struct{}

func NewMatchQueueService() *MatchQueueService {
	return &MatchQueueService{}
}

// getQueueKey 根据mode获取队列key
func (s *MatchQueueService) getQueueKey(mode string) string {
	return MATCH_QUEUE_PREFIX + mode
}

// JoinQueue 加入匹配队列，存储address和publickey
func (s *MatchQueueService) JoinQueue(mode string, address string, publicKey string) error {
	conn := redis.GetGlobalPool().Get()
	defer conn.Close()

	queueKey := s.getQueueKey(mode)

	// 检查是否已在队列
	players, err := s.GetQueue(mode)
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
	log.Infof("玩家 %s (mode: %s) 成功加入匹配队列", address, mode)
	return nil
}

// LeaveQueue 从队列移除address
func (s *MatchQueueService) LeaveQueue(mode string, address string) error {
	conn := redis.GetGlobalPool().Get()
	defer conn.Close()

	queueKey := s.getQueueKey(mode)

	// 检查用户是否已经匹配
	matches, err := db.GetMatchesByAddress(address)
	if err != nil {
		log.Errorf("查询用户匹配记录失败: %v", err)
		return fmt.Errorf("查询匹配状态失败")
	}

	// 检查是否有未完成的匹配
	for _, match := range matches {
		if match.Mode == mode && (match.Status == "matched" || match.Status == "confirmed") {
			return fmt.Errorf("您已经匹配成功，无法离开队列。请先确认或取消匹配")
		}
	}

	// 获取队列中的所有玩家
	players, err := s.GetQueue(mode)
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

	log.Infof("玩家 %s (mode: %s) 已离开匹配队列", address, mode)
	return nil
}

// GetQueue 获取当前队列所有玩家信息
func (s *MatchQueueService) GetQueue(mode string) ([]PlayerInfo, error) {
	conn := redis.GetGlobalPool().Get()
	defer conn.Close()

	queueKey := s.getQueueKey(mode)

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
func (s *MatchQueueService) GetQueueAddresses(mode string) ([]string, error) {
	players, err := s.GetQueue(mode)
	if err != nil {
		return nil, err
	}

	var addresses []string
	for _, player := range players {
		addresses = append(addresses, player.Address)
	}

	return addresses, nil
}

// ProcessMatchmaking 处理匹配逻辑
func (s *MatchQueueService) ProcessMatchmaking(mode string) error {
	// 获取当前队列
	players, err := s.GetQueue(mode)
	if err != nil {
		log.Errorf("获取匹配队列失败: %v", err)
		return err
	}

	// 检查是否达到匹配阈值
	if len(players) < MATCH_THRESHOLD {
		return nil // 人数不足，不进行匹配
	}

	// 选择队列中第一个玩家
	player1 := players[0]

	// 从剩余玩家中随机选择一个进行匹配
	rand.Seed(time.Now().UnixNano())
	player2Index := rand.Intn(len(players)-1) + 1 // 从索引1开始，排除第一个玩家
	player2 := players[player2Index]

	// 创建匹配记录
	matchID := uuid.New().String()

	// 为玩家1创建匹配记录
	match1 := &dao.Match{
		MatchID:   matchID,
		Address:   player1.Address,
		PublicKey: player1.PublicKey,
		Mode:      mode,
		Status:    "matched",
		RoomID:    "", // 初始化为空
	}

	// 为玩家2创建匹配记录
	match2 := &dao.Match{
		MatchID:   matchID,
		Address:   player2.Address,
		PublicKey: player2.PublicKey,
		Mode:      mode,
		Status:    "matched",
		RoomID:    "", // 初始化为空
	}

	// 保存到数据库
	err = db.CreateMatch(match1)
	if err != nil {
		log.Errorf("创建玩家1匹配记录失败: %v", err)
		return err
	}

	err = db.CreateMatch(match2)
	if err != nil {
		log.Errorf("创建玩家2匹配记录失败: %v", err)
		return err
	}

	// 从队列中移除这两个玩家
	err = s.removePlayersFromQueue(mode, []string{player1.Address, player2.Address})
	if err != nil {
		log.Errorf("从队列移除匹配玩家失败: %v", err)
		return err
	}

	log.Infof("成功匹配玩家 %s 和 %s (MatchID: %s, Mode: %s)",
		player1.Address, player2.Address, matchID, mode)

	return nil
}

// removePlayersFromQueue 从队列中移除指定的玩家
func (s *MatchQueueService) removePlayersFromQueue(mode string, addresses []string) error {
	conn := redis.GetGlobalPool().Get()
	defer conn.Close()

	queueKey := s.getQueueKey(mode)

	// 获取队列中的所有玩家
	players, err := s.GetQueue(mode)
	if err != nil {
		return err
	}

	// 创建要移除的地址集合
	removeSet := make(map[string]bool)
	for _, addr := range addresses {
		removeSet[addr] = true
	}

	// 重新构建队列，排除要移除的玩家
	_, err = conn.Do("DEL", queueKey)
	if err != nil {
		return err
	}

	// 重新添加其他玩家
	for _, player := range players {
		if !removeSet[player.Address] {
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

	return nil
}
