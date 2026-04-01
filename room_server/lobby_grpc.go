package roomserver

import (
	"context"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/game"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type lobbySettlementGRPC struct {
	client proto.LobbySettlementServiceClient
}

func (s *lobbySettlementGRPC) GameResultSettlement(evt *types.GameCompletedEvent) error {
	if s.client == nil || evt == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := s.client.NotifyGameCompleted(ctx, &proto.NotifyGameCompletedRequest{GameId: uint32(evt.GameID)})
	return err
}

var _ game.GameResultSettler = (*lobbySettlementGRPC)(nil)

// connectLobby dials ele-lobbyserver for settlement notifications (call after gRPC listener is up).
func (s *Service) connectLobby(ctx context.Context) error {
	if s.cfg.LobbyServerAddress == "" {
		return fmt.Errorf("lobby-server-address is required")
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
	}
	const maxAttempts = 120
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		conn, err := grpc.NewClient(s.cfg.LobbyServerAddress, opts...)
		if err == nil {
			s.lobbyConn = conn
			s.gameSvc.SetGameResultSettler(&lobbySettlementGRPC{client: proto.NewLobbySettlementServiceClient(conn)})
			log.Infow("room: connected to lobby (settlement)", "addr", s.cfg.LobbyServerAddress)
			return nil
		}
		lastErr = err
		log.Warnw("room: dial lobby, retrying", "addr", s.cfg.LobbyServerAddress, "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("dial lobby at %s: %w", s.cfg.LobbyServerAddress, lastErr)
}
