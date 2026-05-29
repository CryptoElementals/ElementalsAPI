package client

import (
	"github.com/CryptoElementals/common/config"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
)

// LobbyClientForType returns the lobby client for the user's server type.
func LobbyClientForType(serverType string) pb.LobbyServiceClient {
	if cl := GetLobbyServiceClient(config.NormalizeServerType(serverType)); cl != nil {
		return cl
	}
	return GetGlobalLobbyClient()
}

// RoomClientForType returns the room client for the user's server type.
func RoomClientForType(serverType string) pb.RoomServiceClient {
	if cl := GetRoomServiceClient(config.NormalizeServerType(serverType)); cl != nil {
		return cl
	}
	return GetGlobalRpcClient()
}

// EventStreamForType returns the Redis event stream for the user's server type.
func EventStreamForType(serverType string) stream.Stream {
	if st := GetEventStream(config.NormalizeServerType(serverType)); st != nil {
		return st
	}
	return GetGlobalEventStream()
}

// ConfiguredTypeEnvKeys returns environment keys for trial and normal when initialized.
func ConfiguredTypeEnvKeys() []string {
	keys := make([]string, 0, len(config.RequiredEnvironmentNames()))
	for _, name := range config.RequiredEnvironmentNames() {
		if _, ok := GetClientContext(name); ok {
			keys = append(keys, name)
		}
	}
	return keys
}
