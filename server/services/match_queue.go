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
	MATCH_THRESHOLD    = 3              // 匹配阈值（3人）
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
		log.Errorf("Failed to get match queue: %v", err)
		return err
	}

	for _, player := range players {
		if player.Address == address {
			return fmt.Errorf("Player is already in match queue")
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
		log.Errorf("Failed to serialize player info: %v", err)
		return err
	}

	// 加入队列（队尾）
	_, err = conn.Do("RPUSH", queueKey, playerData)
	if err != nil {
		log.Errorf("Failed to join match queue: %v", err)
		return err
	}
	log.Infof("Player %s (mode: %s) successfully joined match queue", address, mode)
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
		log.Errorf("Failed to query user match records: %v", err)
		return fmt.Errorf("Failed to query match status")
	}

	// 检查是否有未完成的匹配
	for _, match := range matches {
		if match.Mode == mode && ((match.Player1Status == "matched" || match.Player1Status == "confirmed") ||
			(match.Player2Status == "matched" || match.Player2Status == "confirmed")) {
			return fmt.Errorf("You have been matched successfully and cannot leave the queue. Please confirm or cancel the match first")
		}
	}

	// 获取队列中的所有玩家
	players, err := s.GetQueue(mode)
	if err != nil {
		log.Errorf("Failed to get match queue: %v", err)
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
		return fmt.Errorf("Player is not in match queue")
	}

	// 重新构建队列，排除要移除的玩家
	_, err = conn.Do("DEL", queueKey)
	if err != nil {
		log.Errorf("Failed to clear queue: %v", err)
		return err
	}

	// 重新添加其他玩家
	for _, player := range players {
		if player.Address != address {
			playerData, err := json.Marshal(player)
			if err != nil {
				log.Errorf("Failed to serialize player info: %v", err)
				continue
			}
			_, err = conn.Do("RPUSH", queueKey, playerData)
			if err != nil {
				log.Errorf("Failed to re-add player to queue: %v", err)
			}
		}
	}

	log.Infof("Player %s (mode: %s) has left match queue", address, mode)
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
			log.Errorf("Failed to deserialize player info: %v", err)
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
		log.Errorf("Failed to get match queue: %v", err)
		return err
	}

	// 添加队列状态日志
	log.Infof("Matchmaking task running for mode: %s, current queue size: %d", mode, len(players))

	// 检查是否达到匹配阈值
	if len(players) < MATCH_THRESHOLD {
		log.Debugf("Queue size %d is less than threshold %d, no matching needed for mode: %s", len(players), MATCH_THRESHOLD, mode)
		return nil // 人数不足，不进行匹配
	}

	// 记录队列中的玩家信息
	playerAddresses := make([]string, len(players))
	for i, player := range players {
		playerAddresses[i] = player.Address
	}
	log.Infof("Found %d players in queue for mode %s: %v", len(players), mode, playerAddresses)

	// 选择队列中第一个玩家
	player1 := players[0]

	// 从剩余玩家中随机选择一个进行匹配
	rand.Seed(time.Now().UnixNano())
	player2Index := rand.Intn(len(players)-1) + 1 // 从索引1开始，排除第一个玩家
	player2 := players[player2Index]

	log.Infof("Attempting to match players: %s (index 0) and %s (index %d) for mode: %s",
		player1.Address, player2.Address, player2Index, mode)

	// 创建匹配记录（一行记录包含两个玩家）
	matchID := uuid.New().String()

	match := &dao.Match{
		MatchID: matchID,
		Mode:    mode,
		RoomID:  "", // 初始化为空

		// 玩家1信息
		Player1Address:     player1.Address,
		Player1TempAddress: player1.Address, // 临时地址暂时使用原地址
		Player1Status:      "matched",

		// 玩家2信息
		Player2Address:     player2.Address,
		Player2TempAddress: player2.Address, // 临时地址暂时使用原地址
		Player2Status:      "matched",
	}

	// 保存到数据库
	err = db.CreateMatch(match)
	if err != nil {
		log.Errorf("Failed to create match record: %v", err)
		return err
	}
	log.Infof("Created match record for players: %s and %s (MatchID: %s)", player1.Address, player2.Address, matchID)

	// 从队列中移除这两个玩家
	err = s.removePlayersFromQueue(mode, []string{player1.Address, player2.Address})
	if err != nil {
		log.Errorf("Failed to remove matched players from queue: %v", err)
		return err
	}

	log.Infof("Successfully matched players %s and %s (MatchID: %s, Mode: %s)",
		player1.Address, player2.Address, matchID, mode)

	// 记录匹配后的队列状态
	remainingPlayers, err := s.GetQueue(mode)
	if err != nil {
		log.Errorf("Failed to get remaining queue after matching: %v", err)
	} else {
		log.Infof("Remaining players in queue for mode %s: %d", mode, len(remainingPlayers))
		if len(remainingPlayers) > 0 {
			remainingAddresses := make([]string, len(remainingPlayers))
			for i, player := range remainingPlayers {
				remainingAddresses[i] = player.Address
			}
			log.Infof("Remaining player addresses: %v", remainingAddresses)
		}
	}

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
				log.Errorf("Failed to serialize player info: %v", err)
				continue
			}
			_, err = conn.Do("RPUSH", queueKey, playerData)
			if err != nil {
				log.Errorf("Failed to re-add player to queue: %v", err)
			}
		}
	}

	return nil
}
