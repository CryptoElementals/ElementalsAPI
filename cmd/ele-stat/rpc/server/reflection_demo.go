package server

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

// DemoReflectionServiceDiscovery demonstrate how to use gRPC reflection to discover services
func DemoReflectionServiceDiscovery(serverAddr string) {
	log.Printf("=== gRPC Reflection Service Discovery Demo ===")
	log.Printf("Connecting to server: %s", serverAddr)

	// Connect to the gRPC server
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}
	defer conn.Close()

	// Create reflection client
	reflectionClient := grpc_reflection_v1alpha.NewServerReflectionClient(conn)

	// Create stream for reflection
	ctx := context.Background()
	stream, err := reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		log.Printf("Failed to create reflection stream: %v", err)
		return
	}
	defer stream.CloseSend()

	// Request list of services
	request := &grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_ListServices{},
	}

	if err := stream.Send(request); err != nil {
		log.Printf("Failed to send reflection request: %v", err)
		return
	}

	// Receive response
	response, err := stream.Recv()
	if err != nil {
		log.Printf("Failed to receive reflection response: %v", err)
		return
	}

	// Parse services from response
	if listResponse := response.GetListServicesResponse(); listResponse != nil {
		log.Printf("Discovered %d services:", len(listResponse.Service))

		for i, service := range listResponse.Service {
			log.Printf("  %d. %s", i+1, service.Name)
		}
	} else {
		log.Printf("No services found in reflection response")
	}

	log.Printf("=== Demo completed ===")
}

// GetServicesViaReflection get list of services using gRPC reflection
func GetServicesViaReflection(serverAddr string) ([]string, error) {
	// Connect to the gRPC server
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Create reflection client
	reflectionClient := grpc_reflection_v1alpha.NewServerReflectionClient(conn)

	// Create stream for reflection
	ctx := context.Background()
	stream, err := reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %v", err)
	}
	defer stream.CloseSend()

	// Request list of services
	request := &grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_ListServices{},
	}

	if err := stream.Send(request); err != nil {
		return nil, fmt.Errorf("failed to send reflection request: %v", err)
	}

	// Receive response
	response, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive reflection response: %v", err)
	}

	// Parse services from response
	var services []string
	if listResponse := response.GetListServicesResponse(); listResponse != nil {
		for _, service := range listResponse.Service {
			services = append(services, service.Name)
		}
	}

	return services, nil
}

// GetServiceMethodsViaReflection get methods for a specific service using gRPC reflection
func GetServiceMethodsViaReflection(serverAddr, serviceName string) ([]string, error) {
	// Connect to the gRPC server
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Create reflection client
	reflectionClient := grpc_reflection_v1alpha.NewServerReflectionClient(conn)

	// Create stream for reflection
	ctx := context.Background()
	stream, err := reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %v", err)
	}
	defer stream.CloseSend()

	// Request file containing the service
	request := &grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: serviceName,
		},
	}

	if err := stream.Send(request); err != nil {
		return nil, fmt.Errorf("failed to send reflection request: %v", err)
	}

	// Receive response
	response, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive reflection response: %v", err)
	}

	// Parse file descriptor from response
	if fileResponse := response.GetFileDescriptorResponse(); fileResponse != nil {
		// Note: This is a simplified example
		// In a real implementation, you would parse the file descriptor
		// to extract method information
		log.Printf("File descriptor received for service: %s", serviceName)
		return []string{"Method details would be parsed from file descriptor"}, nil
	}

	return nil, fmt.Errorf("no file descriptor found for service: %s", serviceName)
}
