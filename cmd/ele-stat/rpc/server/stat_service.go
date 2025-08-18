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
