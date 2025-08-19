package server

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/cmd/ele-stat/proto"
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

// UpdatePlayerStats implement update player statistics RPC
func (s *StatService) UpdatePlayerStats(ctx context.Context, req *proto.UpdatePlayerStatsRequest) (*proto.UpdatePlayerStatsResponse, error) {
	log.Infof("Update player stats request received from client: %s, player: %s", req.ClientId, req.PlayerId)

	// Calculate new statistics
	newLevel := req.Level
	newExperience := req.Experience
	newWinRate := req.WinRate

	// Simple level up logic (every 1000 experience = 1 level)
	if newExperience >= 1000 {
		levelIncrease := newExperience / 1000
		newLevel += levelIncrease
		newExperience = newExperience % 1000
	}

	// Recalculate win rate based on wins, losses, and draws
	totalGames := req.Wins + req.Losses + req.Draws
	if totalGames > 0 {
		newWinRate = float32(req.Wins) / float32(totalGames) * 100.0
	}

	// Create response
	response := &proto.UpdatePlayerStatsResponse{
		Status:        "OK",
		Message:       "Player statistics updated successfully",
		PlayerId:      req.PlayerId,
		NewLevel:      newLevel,
		NewExperience: newExperience,
		NewWinRate:    newWinRate,
		Timestamp:     time.Now().Unix(),
		Updated:       true,
	}

	log.Infof("Player %s stats updated: Level %d -> %d, Experience %d -> %d, Win Rate %.2f%% -> %.2f%%",
		req.PlayerId, req.Level, newLevel, req.Experience, newExperience, req.WinRate, newWinRate)

	return response, nil
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
