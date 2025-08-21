package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceRegistry(t *testing.T) {
	// This test is no longer relevant as we removed the service registry
	// We now use gRPC reflection to discover services
	t.Skip("Service registry test skipped - using gRPC reflection instead")
}

func TestStatServerWithServiceRegistry(t *testing.T) {
	// Test server creation
	config := DefaultServerConfig(30012)
	server, err := NewStatServer(config)
	assert.NoError(t, err)
	assert.NotNil(t, server)

	// Test server info
	serverInfo := server.GetServerInfo()
	assert.Equal(t, uint32(30012), serverInfo["port"])
	assert.Contains(t, serverInfo, "services")
	assert.Contains(t, serverInfo["services"], "Use gRPC reflection to discover services dynamically")
}

func TestGetRegisteredServices(t *testing.T) {
	// Test that GetRegisteredServices now dynamically discovers services
	services := GetRegisteredServices()
	assert.NotEmpty(t, services)

	// Should contain actual service names from the running server
	assert.Contains(t, services, "stat.StatService")
	assert.Contains(t, services, "grpc.reflection.v1alpha.ServerReflection")

	for i, service := range services {
		t.Logf("services[%d]: %v", i, service)
	}
}

func TestGetServicesDynamically(t *testing.T) {
	// Test that GetServicesDynamically function exists and can be called
	// Note: This test won't actually connect to a server, just verifies the function exists
	_, err := GetServicesDynamically("localhost:9999") // Use non-existent port
	assert.Error(t, err)                               // Should fail to connect
	// The error message varies depending on the exact failure, so just check that it's an error
	assert.Contains(t, err.Error(), "connection")
}
