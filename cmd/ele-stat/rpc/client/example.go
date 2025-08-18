package client

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
)

// ExampleUsage demonstrate client usage examples
func ExampleUsage() {
	// Create client manager
	manager := NewClientManager()
	defer manager.CloseAllClients()

	// Connect to server
	serverAddr := "localhost:30011"
	client, err := manager.GetOrCreateClient(serverAddr)
	if err != nil {
		log.Errorf("Failed to connect to server: %v", err)
		return
	}

	// Perform health check directly
	resp, err := client.HealthCheck("example-client")
	if err != nil {
		log.Errorf("Health check failed: %v", err)
		return
	}

	log.Infof("Health check successful: Status=%s, Uptime=%s, Message=%s",
		resp.Status, resp.Uptime, resp.Message)
}

// ExampleConnectionManagement demonstrate connection management examples
func ExampleConnectionManagement() {
	manager := NewClientManager()
	defer manager.CloseAllClients()

	// Connect to multiple servers
	servers := []string{
		"localhost:30011",
		"localhost:30012",
		"localhost:30013",
	}

	clients := make(map[string]*StatClient)
	for _, server := range servers {
		client, err := manager.GetOrCreateClient(server)
		if err != nil {
			log.Warnf("Failed to connect to %s: %v", server, err)
			continue
		}
		clients[server] = client
		log.Infof("Connected to %s", server)
	}

	// Get all client status
	status := manager.GetClientStatus()
	for server, state := range status {
		log.Infof("Server %s: %s", server, state)
	}

	// Perform health check on all servers
	healthResults := manager.HealthCheckAll()
	for server, result := range healthResults {
		log.Infof("Server %s health: %s", server, result.Status)
	}

	// Use clients variable to avoid unused warning
	log.Infof("Total connected clients: %d", len(clients))
}

// ExampleContextUsage demonstrate context usage examples
func ExampleContextUsage() {
	manager := NewClientManager()
	defer manager.CloseAllClients()

	serverAddr := "localhost:30011"
	client, err := manager.GetOrCreateClient(serverAddr)
	if err != nil {
		log.Errorf("Failed to connect: %v", err)
		return
	}

	// Use timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait for connection to be ready
	err = client.WaitForConnection(ctx)
	if err != nil {
		log.Errorf("Failed to wait for connection: %v", err)
		return
	}

	log.Info("Connection is ready")

	// Check connection status
	if client.IsConnected() {
		log.Info("Client is connected")
	} else {
		log.Warn("Client is not connected")
	}

	// Print client info
	PrintClientInfo(client)
}

// PrintClientInfo print client information
func PrintClientInfo(client *StatClient) {
	fmt.Printf("Client Configuration:\n")
	fmt.Printf("  Server Address: %s\n", client.GetConfig().ServerAddr)
	fmt.Printf("  Connection State: %s\n", client.GetConnectionState())
	fmt.Printf("  Is Connected: %t\n", client.IsConnected())
	fmt.Printf("  Connection Timeout: %v\n", client.GetConfig().ConnectionTimeout)
	fmt.Printf("  Keepalive Time: %v\n", client.GetConfig().KeepaliveTime)
	fmt.Printf("  Max Receive Message Size: %d bytes\n", client.GetConfig().MaxReceiveMessageSize)
	fmt.Printf("  Max Send Message Size: %d bytes\n", client.GetConfig().MaxSendMessageSize)
}
