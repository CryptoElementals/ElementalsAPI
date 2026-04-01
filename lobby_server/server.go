package lobbyserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/lobby_server/roomclient"
	"github.com/CryptoElementals/common/lobby_server/worker/queue"
	"github.com/CryptoElementals/common/lobby_server/worker/turnament"
	rpcserver "github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Service is the lobby process: queue, tournament worker, PubSub, and gRPC to room.
type Service struct {
	ctx          context.Context
	cfg          *config.LobbyServerConfig
	grpcServer   *grpc.Server
	pubsub       *rpcserver.PubSub
	queueSvc     *queue.Service
	tournSvc     *turnament.TournamentQueueService
	grpcHandlers *GRPCServices
	roomConn     *grpc.ClientConn
}

// New constructs a lobby server. Call Start after DB/redis are initialized.
func New(ctx context.Context, cfg *config.LobbyServerConfig) (*Service, error) {
	s := &Service{ctx: ctx, cfg: cfg}
	s.pubsub = rpcserver.NewPubSub()

	var c cache.Cache
	var err error
	if cfg.IsDevelop {
		c = cache.NewMemCache()
	} else {
		c, err = cache.NewRedisCache()
		if err != nil {
			return nil, err
		}
	}

	argsTemplate, err := db.LoadRoomServerGameArgs(cfg.GameArgsID)
	if err != nil {
		return nil, fmt.Errorf("game_args template (game-args-id=%d): %w", cfg.GameArgsID, err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
	}
	conn, err := dialRoomWorkerWithRetry(ctx, cfg.RoomServerAddress, opts)
	if err != nil {
		return nil, err
	}
	s.roomConn = conn
	rw := proto.NewRoomWorkerServiceClient(conn)
	gc := &roomclient.GameCreator{Client: rw}

	s.queueSvc = queue.NewService(ctx, s.pubsub, c, gc,
		cfg.MinTokenToJoinQueue,
		argsTemplate.ConfirmationTimeout,
		argsTemplate.GameContinueTimeout,
		argsTemplate.GameContinueTimeoutRedundancy,
		cfg.BotWaitTime,
		cfg.StatServiceEndpoint,
	)
	s.tournSvc = turnament.NewTournamentQueueService(ctx, gc, cfg.MinTokenToJoinQueue)
	s.grpcHandlers = NewGRPCServices(s.queueSvc, s.tournSvc, rw)
	s.pubsub.SetBotHooks(&lobbyPubSubBots{q: s.queueSvc})

	s.grpcServer = grpc.NewServer()
	proto.RegisterLobbyServiceServer(s.grpcServer, s.grpcHandlers)
	proto.RegisterLobbySettlementServiceServer(s.grpcServer, s.grpcHandlers)
	proto.RegisterPubSubServiceServer(s.grpcServer, s.pubsub)

	return s, nil
}

func dialRoomWorkerWithRetry(ctx context.Context, addr string, opts []grpc.DialOption) (*grpc.ClientConn, error) {
	const maxAttempts = 60
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		conn, err := grpc.NewClient(addr, opts...)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		log.Warnw("lobby: dial room worker, retrying", "addr", addr, "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("dial room at %s after %d attempts: %w", addr, maxAttempts, lastErr)
}

// Start queue, tournament coordinator, and gRPC listener.
func (s *Service) Start() error {
	if err := s.queueSvc.Start(); err != nil {
		return err
	}
	s.tournSvc.Start()
	return s.startListener()
}

func (s *Service) startListener() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.ListenPort))
	if err != nil {
		return err
	}
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Fatalf("lobby grpc serve failed: %v", err)
		}
	}()
	return nil
}

// Stop shuts down background workers and gRPC.
func (s *Service) Stop() {
	s.tournSvc.Stop()
	s.queueSvc.Stop()
	s.pubsub.Stop()
	s.grpcServer.GracefulStop()
	if s.roomConn != nil {
		_ = s.roomConn.Close()
	}
}
