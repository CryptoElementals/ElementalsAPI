# gRPC Client Usage Guide

This directory contains a complete gRPC client implementation that supports connecting to statistics servers and performing various operations.

## File Structure

- `client.go` - Complete client implementation with connection management and health check functionality
- `factory.go` - Client factory and manager for multiple connections
- `example.go` - Usage examples and demonstration code
- `client_test.go` - Comprehensive test suite

## Main Features

### 1. Basic Client (StatClient)

```go
// Create client configuration
config := client.DefaultClientConfig("localhost:30011")

// Create client
client, err := client.NewStatClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Check connection status
if client.IsConnected() {
    log.Info("Client is connected")
}
```

### 2. Health Check (Integrated into StatClient)

```go
// Execute health check directly on StatClient
resp, err := client.HealthCheck("my-client")
if err != nil {
    log.Error(err)
    return
}

log.Infof("Health status: %s", resp.Status)
```

### 3. Client Manager (ClientManager)

```go
// Create manager
manager := client.NewClientManager()
defer manager.CloseAllClients()

// Get or create client
client, err := manager.GetOrCreateClient("localhost:30011")
if err != nil {
    log.Fatal(err)
}

// Get all client status
status := manager.GetClientStatus()
for server, state := range status {
    log.Infof("Server %s: %s", server, state)
}
```

## Configuration Options

### ClientConfig

- `ServerAddr` - Server address
- `ConnectionTimeout` - Connection timeout
- `KeepaliveTime` - Keepalive time
- `KeepaliveTimeout` - Keepalive timeout
- `MaxReceiveMessageSize` - Maximum receive message size
- `MaxSendMessageSize` - Maximum send message size

### Default Configuration

```go
config := client.DefaultClientConfig("localhost:30011")
// Connection timeout: 10 seconds
// Keepalive time: 30 seconds
// Keepalive timeout: 5 seconds
// Maximum message size: 4MB
```

## Advanced Features

### 1. Retry Mechanism

```go
// Health check with retry
resp, err := client.HealthCheckWithRetry("client-id", 3)
```

### 2. Batch Operations

```go
// Batch health check
clientIDs := []string{"client1", "client2", "client3"}
results, err := healthClient.BatchHealthCheck(ctx, clientIDs)
```

### 3. Connection Management

```go
// Wait for connection to be ready
err = client.WaitForConnection(ctx)

// Get connection status
state := client.GetConnectionState()
```

## Error Handling

The client includes a robust error handling mechanism:

- Automatic retry on connection failure
- Timeout control
- Connection status monitoring
- Graceful shutdown

## Usage Examples

See the complete usage examples in the `example.go` file:

```go
// Basic usage example
client.ExampleUsage()

// Connection management example
client.ExampleConnectionManagement()

// Context usage example
client.ExampleContextUsage()
```

## Notes

1. **Resource Management**: Remember to call the `Close()` method after use
2. **Connection Pool**: Use `ClientManager` to manage multiple connections
3. **Timeout Control**: Set appropriate timeout values to avoid long waits
4. **Error Handling**: Check all returned errors and handle appropriately

## Extending New Services

When adding new RPC services:

1. Add a new `.proto` file in the `proto/` directory
2. Run `make proto` to generate code
3. Create the corresponding client implementation
4. Register the service in `factory.go`

## Testing

```bash
# Compile tests
go build ./...

# Run examples
go run example.go
```

# Client Test Runner

A flexible Go test runner script specifically designed for the `rpc/client` package.

## 🚀 Quick Start

### Basic Usage
```bash
# Run all tests
./run_tests.sh

# Detailed mode for all tests
./run_tests.sh -v

# Run specific tests
./run_tests.sh TestRealGRPCConnection

# Run multiple specific tests
./run_tests.sh TestDefaultClientConfig TestNewStatClient
```

## 📋 Available Options

| Option | Long Option | Description |
|------|--------|------|
| `-h` | `--help` | Show help information |
| `-v` | `--verbose` | Detailed output mode |
| `-c` | `--coverage` | Generate coverage report |
| `-r` | `--race` | Enable race detection |
| `-s` | `--short` | Only run short-time tests |
| `-l` | `--list` | List all available test cases |
| `-f` | `--filter PATTERN` | Filter tests by pattern |
| `-m` | `--mode MODE` | Run mode |
| `-t` | `--timeout DURATION` | Set test timeout |
| `-o` | `--output FILE` | Save output to file |
| `-j` | `--jobs N` | Number of tests to run in parallel |

## 🎯 Run Modes

### `all` (default)
Run all test cases

### `unit`
Only run unit tests (simulated mode)
- Does not depend on external services
- Tests basic client functionality
- Includes configuration validation, structure tests, etc.

### `integration`
Only run integration tests (real connection)
- Requires stat server to be running
- Tests interaction with the real server
- Includes connection tests, health checks, etc.

### `real`
Only run real connection tests
- Requires stat server to be running
- Tests network connection and server response

### `custom`
Run custom test cases
- Can specify specific test names
- Supports regex filtering

## 🔍 Test Case Categories

### Basic Functionality Tests
- `TestDefaultClientConfig` - Default configuration test
- `TestNewStatClient` - Client creation test
- `TestClientConfig_Validation` - Configuration validation test

### Health Check Tests
- `TestHealthClient_HealthCheck` - Basic health check
- `TestHealthClient_HealthCheckWithRetry` - Retry mechanism test
- `TestHealthClient_BatchHealthCheck` - Batch health check
- `TestHealthRequest_Structure` - Request structure test
- `TestHealthResponse_Structure` - Response structure test

### Client Manager Tests
- `TestClientManager_Creation` - Manager creation test
- `TestClientManager_GetClientStatus` - Status retrieval test
- `TestClientManager_CloseAllClients` - Close all clients test

### Real Connection Tests
- `TestRealGRPCConnection` - Real gRPC connection test
- `TestClientManagerRealConnection` - Manager real connection test
- `TestBatchHealthCheckRealServer` - Batch health check real server test

## 💡 Usage Examples

### Development Stage
```bash
# Quick run unit tests
./run_tests.sh -m unit

# Run specific functionality tests
./run_tests.sh -f TestHealth

# Enable race detection
./run_tests.sh -r
```

### Integration Tests
```bash
# Ensure stat server is running
./run_tests.sh -m integration

# Only test real connection
./run_tests.sh -m real
```

### Quality Assurance
```bash
# Generate coverage report
./run_tests.sh -c -v

# Save test results to file
./run_tests.sh -v -o test_results.log

# Run tests in parallel
./run_tests.sh -j 8
```

### Debug Specific Issues
```bash
# Run specific test cases
./run_tests.sh TestRealGRPCConnection

# Run related tests
./run_tests.sh -f TestClientManager

# Detailed output + coverage
./run_tests.sh -v -c TestHealthClient_HealthCheck
```

## 📊 Coverage Report

After enabling coverage options (`-c`), the script will:
1. Generate `coverage.out` file (raw data)
2. Generate `coverage.html` file (HTML report)
3. Display coverage statistics on the console

## 🛠️ Troubleshooting

### Common Issues

1. **Permission Errors**
   ```bash
   chmod +x rpc/client/run_tests.sh
   ```

2. **Missing Test Dependencies**
   ```bash
   go mod tidy
   go get -t ./...
   ```

3. **Real connection tests failing**
   - Ensure stat server is running
   - Check if port 30011 is accessible
   - Verify network connection

4. **Test Timeout**
   ```bash
   ./run_tests.sh -t 10m
   ```

## 🔧 Custom Configuration

The script supports the following environment variables:
- `GOTEST_TIMEOUT` - Test timeout
- `GOMAXPROCS` - Number of parallel tests

## 📝 Notes

1. **Real connection tests** require stat server to be running on `localhost:30011`
2. **Coverage report** generates temporary files, the script will clean them up
3. **Race detection** increases test runtime
4. **Detailed mode** shows all test outputs and logs

## 🤝 Contributing

If you want to add new test cases or features, please:
1. Follow the existing test naming conventions
2. Update the corresponding test categories
3. Test various run modes of the script
4. Update this document
