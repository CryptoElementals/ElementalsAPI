package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/redis"
)

// ConnectionHealthMonitor probes gRPC and Redis used by the event pipeline.
type ConnectionHealthMonitor struct {
	mu              sync.RWMutex
	lastHealthCheck time.Time
	isHealthy       bool
	failureCount    int
	successCount    int
	totalChecks     int
}

// NewConnectionHealthMonitor creates a connection health monitor.
func NewConnectionHealthMonitor() *ConnectionHealthMonitor {
	return &ConnectionHealthMonitor{
		lastHealthCheck: time.Now(),
		isHealthy:       false,
	}
}

// CheckHealth runs a health check and returns whether dependencies are reachable.
func (chm *ConnectionHealthMonitor) CheckHealth() bool {
	chm.mu.Lock()
	defer chm.mu.Unlock()

	chm.totalChecks++
	chm.lastHealthCheck = time.Now()

	if GetGlobalRpcClient() == nil {
		chm.isHealthy = false
		chm.failureCount++
		log.Warnf("connection health check failed: gRPC room client not initialized")
		return false
	}

	for _, env := range []string{dao.ServerTypeTrial, dao.ServerTypeNormal} {
		if err := pingNamedPool(env); err != nil {
			chm.isHealthy = false
			chm.failureCount++
			log.Warnf("connection health check failed: redis %q: %v", env, err)
			return false
		}
	}

	chm.isHealthy = true
	chm.successCount++
	log.Debugf("connection health check succeeded")
	return true
}

func pingNamedPool(name string) error {
	op, err := redis.Pool(name)
	if err != nil {
		return err
	}
	return op.Ping()
}

// GetHealthStats returns monitor statistics.
func (chm *ConnectionHealthMonitor) GetHealthStats() map[string]interface{} {
	chm.mu.RLock()
	defer chm.mu.RUnlock()

	successRate := float64(0)
	if chm.totalChecks > 0 {
		successRate = float64(chm.successCount) / float64(chm.totalChecks) * 100
	}

	return map[string]interface{}{
		"is_healthy":       chm.isHealthy,
		"last_check":       chm.lastHealthCheck.Format("2006-01-02 15:04:05"),
		"total_checks":     chm.totalChecks,
		"success_count":    chm.successCount,
		"failure_count":    chm.failureCount,
		"success_rate_pct": successRate,
		"checked_pools":    []string{dao.ServerTypeTrial, dao.ServerTypeNormal},
	}
}

// IsHealthy returns the last recorded health status.
func (chm *ConnectionHealthMonitor) IsHealthy() bool {
	chm.mu.RLock()
	defer chm.mu.RUnlock()
	return chm.isHealthy
}

var (
	globalHealthMonitor *ConnectionHealthMonitor
	healthMonitorOnce   sync.Once
)

// GetGlobalHealthMonitor returns the shared connection health monitor.
func GetGlobalHealthMonitor() *ConnectionHealthMonitor {
	healthMonitorOnce.Do(func() {
		globalHealthMonitor = NewConnectionHealthMonitor()
	})
	return globalHealthMonitor
}

// PingAllStreamPools returns an error describing the first unreachable stream Redis pool.
func PingAllStreamPools() error {
	for _, env := range []string{dao.ServerTypeTrial, dao.ServerTypeNormal} {
		if err := pingNamedPool(env); err != nil {
			return fmt.Errorf("%s: %w", env, err)
		}
	}
	return nil
}
