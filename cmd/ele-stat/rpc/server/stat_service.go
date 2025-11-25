package server

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/cmd/ele-stat/proto"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
)

// StatService stat check service implementation
type StatService struct {
	proto.UnimplementedStatServiceServer
	startTime time.Time
}

// NewStatService create new stat check service
func NewStatService() *StatService {
	return &StatService{
		startTime: time.Now(),
	}
}

// HealthCheck implement health check RPC
func (s *StatService) HealthCheck(ctx context.Context, req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	uptime := time.Since(s.startTime)

	log.Infof("Health check request received from client: %s", req.ClientId)

	return &proto.HealthCheckResponse{
		Status:    "OK",
		Uptime:    formatDuration(uptime),
		Timestamp: time.Now().Unix(),
		Message:   "Health check completed successfully",
	}, nil
}

// formatDuration format duration
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// UpdatePlayerStats implement update player statistics RPC
func (s *StatService) UpdatePlayerStats(ctx context.Context, req *proto.UpdatePlayerStatsRequest) (*proto.UpdatePlayerStatsResponse, error) {
	log.Infof("Update player stats request received, count: %d, player ids: %v", len(req.PlayerIds), req.PlayerIds)

	if len(req.PlayerIds) == 0 {
		return &proto.UpdatePlayerStatsResponse{
			Ok:      false,
			Message: "No player ids provided",
		}, nil
	}

	// 调用数据库增量更新函数
	userStats, err := db.UpdateUserStatByAddresses(req.PlayerIds)

	if err != nil {
		// 操作失败
		log.Errorf("Failed to update player stats for player ids %v: %v", req.PlayerIds, err)
		return &proto.UpdatePlayerStatsResponse{
			Ok:      false,
			Message: fmt.Sprintf("Failed to update player stats: %v", err),
		}, nil
	}

	cardStats, err := db.UpdateCardStatByAddresses(req.PlayerIds) // req.PlayerAddresses contains player IDs as strings
	if err != nil {
		log.Errorf("Failed to update card stats for player ids %v: %v", req.PlayerIds, err)
		return &proto.UpdatePlayerStatsResponse{
			Ok:      false,
			Message: fmt.Sprintf("Failed to update card stats for player ids %v: %v", req.PlayerIds, err),
		}, nil
	}

	// 操作成功
	log.Infof("Successfully updated player stats for %d players, card stats count: %d", len(userStats), len(cardStats))
	return &proto.UpdatePlayerStatsResponse{
		Ok:      true,
		Message: fmt.Sprintf("Successfully processed %d players, card stats count: %d", len(userStats), len(cardStats)),
	}, nil
}
