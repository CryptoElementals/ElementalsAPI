package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
)

// ClientManager client manager
type ClientManager struct {
	clients map[string]*StatClient
	mutex   sync.RWMutex
}

// NewClientManager create client manager
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*StatClient),
	}
}

// GetOrCreateClient get or create client
func (cm *ClientManager) GetOrCreateClient(serverAddr string) (*StatClient, error) {
	cm.mutex.RLock()
	if client, exists := cm.clients[serverAddr]; exists && client.IsConnected() {
		cm.mutex.RUnlock()
		return client, nil
	}
	cm.mutex.RUnlock()

	// Create new client
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check again to avoid duplicate creation
	if client, exists := cm.clients[serverAddr]; exists && client.IsConnected() {
		return client, nil
	}

	// Create new client
	config := DefaultClientConfig(serverAddr)
	client, err := NewStatClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for %s: %v", serverAddr, err)
	}

	cm.clients[serverAddr] = client
	log.Infof("Created new client for server: %s", serverAddr)

	return client, nil
}

// CloseClient close specified client
func (cm *ClientManager) CloseClient(serverAddr string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if client, exists := cm.clients[serverAddr]; exists {
		err := client.Close()
		delete(cm.clients, serverAddr)
		log.Infof("Closed client for server: %s", serverAddr)
		return err
	}

	return nil
}

// CloseAllClients close all clients
func (cm *ClientManager) CloseAllClients() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	var lastErr error
	for serverAddr, client := range cm.clients {
		if err := client.Close(); err != nil {
			log.Errorf("Failed to close client for %s: %v", serverAddr, err)
			lastErr = err
		}
	}

	cm.clients = make(map[string]*StatClient)
	log.Info("All clients closed")
	return lastErr
}

// GetClientStatus get status of all managed clients
func (cm *ClientManager) GetClientStatus() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	status := make(map[string]interface{})
	for name, client := range cm.clients {
		clientStatus := map[string]interface{}{
			"connected":        client.IsConnected(),
			"server_addr":      client.GetConfig().ServerAddr,
			"connection_state": client.GetConnectionState(),
		}

		// Try to get service status if connected
		if client.IsConnected() {
			// Try to perform health check directly
			resp, err := client.HealthCheck("status-check")
			if err != nil {
				clientStatus["service_status"] = "ERROR"
				clientStatus["error"] = err.Error()
			} else {
				clientStatus["service_status"] = resp.Status
				clientStatus["uptime"] = resp.Uptime
				clientStatus["message"] = resp.Message
			}
		} else {
			clientStatus["service_status"] = "DISCONNECTED"
		}

		status[name] = clientStatus
	}

	return status
}

// HealthCheckAll perform health check on all servers
func (cm *ClientManager) HealthCheckAll() map[string]*HealthCheckResponse {
	cm.mutex.RLock()
	clients := make(map[string]*StatClient)
	for addr, client := range cm.clients {
		clients[addr] = client
	}
	cm.mutex.RUnlock()

	results := make(map[string]*HealthCheckResponse)
	for serverAddr, client := range clients {
		resp, err := client.HealthCheck("manager")
		if err != nil {
			results[serverAddr] = &HealthCheckResponse{
				Status:    "ERROR",
				Uptime:    "0s",
				Timestamp: time.Now().Unix(),
				Message:   fmt.Sprintf("Health check failed: %v", err),
			}
		} else {
			results[serverAddr] = resp
		}
	}

	return results
}
