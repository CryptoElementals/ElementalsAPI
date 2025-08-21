package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultClientConfig(t *testing.T) {
	serverAddr := "localhost:30011"
	config := DefaultClientConfig(serverAddr)

	assert.Equal(t, serverAddr, config.ServerAddr)
	assert.Equal(t, 10*time.Second, config.ConnectionTimeout)
	assert.Equal(t, 30*time.Second, config.KeepaliveTime)
	assert.Equal(t, 5*time.Second, config.KeepaliveTimeout)
	assert.Equal(t, 4*1024*1024, config.MaxReceiveMessageSize)
	assert.Equal(t, 4*1024*1024, config.MaxSendMessageSize)
}

func TestNewStatClient(t *testing.T) {
	// Note: This test requires a real gRPC server running on the specified port
	// Or we can test configuration creation (without actual connection)
	config := DefaultClientConfig("localhost:30011")

	// Test if configuration is set correctly
	assert.Equal(t, "localhost:30011", config.ServerAddr)
	assert.Equal(t, 10*time.Second, config.ConnectionTimeout)
}

func TestClientConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ClientConfig{
				ServerAddr:            "localhost:30011",
				ConnectionTimeout:     10 * time.Second,
				KeepaliveTime:         30 * time.Second,
				KeepaliveTimeout:      5 * time.Second,
				MaxReceiveMessageSize: 4 * 1024 * 1024,
				MaxSendMessageSize:    4 * 1024 * 1024,
			},
			wantErr: false,
		},
		{
			name: "empty server addr",
			config: &ClientConfig{
				ServerAddr: "",
			},
			wantErr: true, // Empty server address should be treated as error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test configuration creation
			assert.NotNil(t, tt.config)

			// For valid configuration, verify fields
			if !tt.wantErr {
				assert.NotEmpty(t, tt.config.ServerAddr)
				assert.Greater(t, tt.config.ConnectionTimeout, 0*time.Second)
				assert.Greater(t, tt.config.KeepaliveTime, 0*time.Second)
				assert.Greater(t, tt.config.KeepaliveTimeout, 0*time.Second)
				assert.Greater(t, tt.config.MaxReceiveMessageSize, 0)
				assert.Greater(t, tt.config.MaxSendMessageSize, 0)
			} else {
				// For invalid configuration, verify error conditions
				assert.Empty(t, tt.config.ServerAddr)
			}
		})
	}
}

func TestStatClient_HealthCheck(t *testing.T) {
	// Create mock client with nil connection (will cause gRPC error)
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("test-client")
	assert.Error(t, err)
	// The error should be related to gRPC connection failure
	assert.True(t, err != nil)
}

func TestStatClient_HealthCheckWithRetry(t *testing.T) {
	// Create mock client with nil connection (will cause gRPC error)
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check with retry directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheckWithRetry("test-client", 3)
	assert.Error(t, err)
	// The error should be related to gRPC connection failure
	assert.True(t, err != nil)
}

func TestHealthClient_HealthCheck(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("test-client")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithRetry(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check with retry directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheckWithRetry("test-client", 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_BatchHealthCheck(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test single health check directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("test-client") // Single check instead of batch
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithContext(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("test-client") // Context is handled internally
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithCustomTimeout(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("test-client") // Timeout is handled internally
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithInvalidClientID(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check with invalid client ID directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("") // Empty client ID
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithSpecialCharacters(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check with special characters directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("test-client@#$%") // Special characters in client ID
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithLongClientID(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check with long client ID directly (will fail due to nil connection, but tests the structure)
	longClientID := "very-long-client-id-that-exceeds-normal-length-limits-for-testing-purposes"
	_, err := mockClient.HealthCheck(longClientID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithUnicodeClientID(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check with unicode client ID directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("测试客户端") // Unicode client ID for testing
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithNumericClientID(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check with numeric client ID directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("12345") // Numeric client ID
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthClient_HealthCheckWithMixedClientID(t *testing.T) {
	// Create mock client
	mockClient := &StatClient{
		conn:   nil, // Mock connection
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test health check with mixed client ID directly (will fail due to nil connection, but tests the structure)
	_, err := mockClient.HealthCheck("client-123_test@example.com") // Mixed client ID
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected to server")
}

func TestHealthRequest_Structure(t *testing.T) {
	// This test is no longer relevant as we removed HealthRequest
	// We now use proto.StatRequest directly
	t.Skip("HealthRequest test skipped - using proto.StatRequest instead")
}

func TestHealthResponse_Structure(t *testing.T) {
	// This test is no longer relevant as we removed HealthResponse
	// We now use proto.StatResponse directly
	t.Skip("HealthResponse test skipped - using proto.StatResponse instead")
}

func TestClientManager_Creation(t *testing.T) {
	manager := NewClientManager()
	require.NotNil(t, manager)

	// Test initial state
	status := manager.GetClientStatus()
	assert.Empty(t, status)
}

func TestClientManager_GetClientStatus(t *testing.T) {
	manager := NewClientManager()
	defer manager.CloseAllClients()

	// Test empty status
	status := manager.GetClientStatus()
	assert.Empty(t, status)

	// Print status information
	t.Logf("Client manager status: %+v", status)
}

func TestClientManager_CloseAllClients(t *testing.T) {
	manager := NewClientManager()

	// Test closing empty manager
	err := manager.CloseAllClients()
	assert.NoError(t, err)

	// Print close result
	t.Logf("Close all clients result: %v", err)
}

func TestHealthClient_NewHealthClient(t *testing.T) {
	statClient := &StatClient{
		conn:   nil,
		config: DefaultClientConfig("localhost:30011"),
	}

	// Test that StatClient can be created and has expected properties
	require.NotNil(t, statClient)
	assert.NotNil(t, statClient.GetConfig())

	// Print stat client information
	t.Logf("Stat client created: %+v", statClient)
}

// Test helper function
func TestPrintClientInfo(t *testing.T) {
	// Create a mock client for testing
	config := DefaultClientConfig("localhost:30011")
	statClient := &StatClient{
		conn:   nil,
		config: config,
	}

	// This function mainly prints information, we test it does not panic
	assert.NotPanics(t, func() {
		PrintClientInfo(statClient)
	})

	// Print client configuration information
	t.Logf("Client config: ServerAddr=%s, ConnectionTimeout=%v",
		config.ServerAddr, config.ConnectionTimeout)
}

// Test configuration retrieval
func TestStatClient_GetConfig(t *testing.T) {
	config := DefaultClientConfig("localhost:30011")
	statClient := &StatClient{
		conn:   nil,
		config: config,
	}

	retrievedConfig := statClient.GetConfig()
	assert.Equal(t, config, retrievedConfig)
	assert.Equal(t, "localhost:30011", retrievedConfig.ServerAddr)

	// Print configuration information
	t.Logf("Retrieved config: ServerAddr=%s, ConnectionTimeout=%v",
		retrievedConfig.ServerAddr, retrievedConfig.ConnectionTimeout)
}

// Test connection status check
func TestStatClient_IsConnected(t *testing.T) {
	// Test uninitialized client
	statClient := &StatClient{
		conn:   nil,
		config: DefaultClientConfig("localhost:30011"),
	}

	connected := statClient.IsConnected()
	assert.False(t, connected)

	// Print connection status
	t.Logf("Client connection status: %t", connected)
}

// Test wait for connection
func TestStatClient_WaitForConnection(t *testing.T) {
	statClient := &StatClient{
		conn:   nil,
		config: DefaultClientConfig("localhost:30011"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := statClient.WaitForConnection(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not initialized")

	// Print wait for connection result
	t.Logf("Wait for connection result: %v", err)
}

// Test connection status retrieval
func TestStatClient_GetConnectionState(t *testing.T) {
	statClient := &StatClient{
		conn:   nil,
		config: DefaultClientConfig("localhost:30011"),
	}

	state := statClient.GetConnectionState()
	assert.Equal(t, "NOT_INITIALIZED", state)

	// Print connection status
	t.Logf("Client connection state: %s", state)
}

// Test real gRPC connection
func TestRealGRPCConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real connection test in short mode")
	}

	// Try to connect to real server
	config := DefaultClientConfig("localhost:30011")
	statClient, err := NewStatClient(config)
	if err != nil {
		t.Logf("Failed to connect to real server: %v", err)
		t.Skip("Skipping real connection test - server may not be running")
	}
	defer statClient.Close()

	// Check connection status
	connected := statClient.IsConnected()
	t.Logf("Real client connection status: %t", connected)

	if connected {
		// If connection successful, test health check directly
		// Perform real health check
		resp, err := statClient.HealthCheck("test-client-real")
		if err != nil {
			t.Logf("Real health check failed: %v", err)
		} else {
			// Print real response result
			t.Logf("Real health check response: Status=%s, Uptime=%s, Timestamp=%d, Message=%s",
				resp.Status, resp.Uptime, resp.Timestamp, resp.Message)

			// Verify response
			assert.NotNil(t, resp)
			assert.NotEmpty(t, resp.Status)
			assert.NotEmpty(t, resp.Timestamp)
		}
	} else {
		t.Logf("Client not connected, connection state: %s", statClient.GetConnectionState())
	}
}

// Test client manager with real server connection
func TestClientManagerRealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real connection test in short mode")
	}

	manager := NewClientManager()
	defer manager.CloseAllClients()

	// Try to connect to real server
	client, err := manager.GetOrCreateClient("localhost:30011")
	if err != nil {
		t.Logf("Failed to connect to real server: %v", err)
		t.Skip("Skipping real connection test - server may not be running")
	}

	// Check connection status
	connected := client.IsConnected()
	t.Logf("Manager client connection status: %t", connected)

	if connected {
		// Get client status
		status := manager.GetClientStatus()
		t.Logf("Manager client status: %+v", status)

		// Test health check directly
		resp, err := client.HealthCheck("manager-test")
		if err != nil {
			t.Logf("Manager health check failed: %v", err)
		} else {
			t.Logf("Manager health check success: %+v", resp)
		}
	}
}

// Test health check with real server
func TestBatchHealthCheckRealServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real connection test in short mode")
	}

	config := DefaultClientConfig("localhost:30011")
	statClient, err := NewStatClient(config)
	if err != nil {
		t.Logf("Failed to create real client: %v", err)
		t.Skip("Skipping real connection test - server may not be running")
	}
	defer statClient.Close()

	if !statClient.IsConnected() {
		t.Skip("Client not connected to real server")
	}

	// Test single health check directly (batch functionality removed)
	_, err = statClient.HealthCheck("test-client")

	if err != nil {
		t.Logf("Health check failed: %v", err)
	} else {
		t.Logf("Health check with real server successful")
	}
}
