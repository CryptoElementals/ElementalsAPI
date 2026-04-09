package events

import (
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/client"
)

// ConnectionHealthMonitor 连接健康监控器
type ConnectionHealthMonitor struct {
	mu              sync.RWMutex
	lastHealthCheck time.Time
	isHealthy       bool
	failureCount    int
	successCount    int
	totalChecks     int
}

// NewConnectionHealthMonitor 创建连接健康监控器
func NewConnectionHealthMonitor() *ConnectionHealthMonitor {
	return &ConnectionHealthMonitor{
		lastHealthCheck: time.Now(),
		isHealthy:       false,
	}
}

// CheckHealth 检查连接健康状态
func (chm *ConnectionHealthMonitor) CheckHealth() bool {
	chm.mu.Lock()
	defer chm.mu.Unlock()

	chm.totalChecks++
	chm.lastHealthCheck = time.Now()

	// 获取gRPC客户端
	rpcClient := client.GetGlobalRpcClient()
	if rpcClient == nil {
		chm.isHealthy = false
		chm.failureCount++
		log.Warnf("连接健康检查失败: gRPC客户端未初始化")
		return false
	}

	if err := client.CheckRedisPing(); err != nil {
		chm.isHealthy = false
		chm.failureCount++
		log.Warnf("连接健康检查失败: Redis: %v", err)
		return false
	}

	chm.isHealthy = true
	chm.successCount++
	log.Debugf("连接健康检查成功")
	return true
}

// GetHealthStats 获取健康状态统计
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
	}
}

// IsHealthy 返回当前健康状态
func (chm *ConnectionHealthMonitor) IsHealthy() bool {
	chm.mu.RLock()
	defer chm.mu.RUnlock()
	return chm.isHealthy
}

// 为全局事件管理器添加健康监控
var globalHealthMonitor *ConnectionHealthMonitor
var healthMonitorOnce sync.Once

// GetGlobalHealthMonitor 获取全局健康监控器
func GetGlobalHealthMonitor() *ConnectionHealthMonitor {
	healthMonitorOnce.Do(func() {
		globalHealthMonitor = NewConnectionHealthMonitor()
	})
	return globalHealthMonitor
}
