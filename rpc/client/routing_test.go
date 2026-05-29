package client

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	pb "github.com/CryptoElementals/common/rpc/proto"
)

type stubLobbyClient struct {
	pb.LobbyServiceClient
}

func TestLobbyClientForTypeUsesNamedContext(t *testing.T) {
	stub := &stubLobbyClient{}
	SetLobbyClientForTest(dao.ServerTypeTrial, stub)
	t.Cleanup(func() { _ = CloseClientContext(dao.ServerTypeTrial) })

	if got := LobbyClientForType(dao.ServerTypeTrial); got != stub {
		t.Fatal("expected trial lobby client from named context")
	}
}

func TestRoomClientForTypeUsesNamedContext(t *testing.T) {
	c := getOrCreateClientContext(dao.ServerTypeTrial)
	c.Mutex.Lock()
	c.RpcClient = pb.NewRoomServiceClient(nil) // stub not needed for pointer identity
	roomStub := c.RpcClient
	c.Mutex.Unlock()
	t.Cleanup(func() { _ = CloseClientContext(dao.ServerTypeTrial) })

	if got := RoomClientForType(dao.ServerTypeTrial); got != roomStub {
		t.Fatal("expected trial room client from named context")
	}
}
