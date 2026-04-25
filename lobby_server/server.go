package lobbyserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/CryptoElementals/common/bot_manager"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/lobby_server/roomclient"
	"github.com/CryptoElementals/common/lobby_server/worker/queue"
	tournament "github.com/CryptoElementals/common/lobby_server/worker/tournament"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/middleware"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/stream"
	"github.com/CryptoElementals/common/timer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Service is the lobby process: queue, tournament worker, gRPC to room, and Redis event streams.
type Service struct {
	ctx           context.Context
	cfg           *config.LobbyServerConfig
	grpcServer    *grpc.Server
	queueSvc      *queue.Service
	tournamentSvc *tournament.TournamentQueueService
	grpcHandlers  *GRPCServices
	roomConn      *grpc.ClientConn
	eventStream   stream.Stream
}

// New constructs a lobby server. Call Start after DB/redis are initialized.
func New(ctx context.Context, cfg *config.LobbyServerConfig) (*Service, error) {
	s := &Service{ctx: ctx, cfg: cfg}

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
	rw := proto.NewRoomServiceClient(conn)
	gc := &roomclient.GameCreator{Client: rw}

	st, err := stream.NewRedisStream()
	if err != nil {
		return nil, fmt.Errorf("redis stream: %w", err)
	}
	eventPub := pubsub.NewStreamPublisher(st)
	s.eventStream = st

	botStore, err := bot_manager.NewRedisStore("")
	if err != nil {
		return nil, fmt.Errorf("lobby redis bots: %w", err)
	}
	tokenTh := int64(cfg.MinTokenToJoinQueue)
	if int64(cfg.TournamentCfg.EntryFee) > tokenTh {
		tokenTh = int64(cfg.TournamentCfg.EntryFee)
	}
	botStore.SetTokenInsufficientThreshold(tokenTh)
	queueSvc, err := queue.NewService(ctx, eventPub, botStore, gc,
		cfg.MinTokenToJoinQueue,
		argsTemplate.ConfirmationTimeout,
		argsTemplate.GameContinueTimeout,
		argsTemplate.GameContinueTimeoutRedundancy,
		cfg.BotWaitTime,
		cfg.BotRegistryFreshnessSec,
		cfg.StatServiceEndpoint,
	)
	if err != nil {
		return nil, fmt.Errorf("queue service: %w", err)
	}
	s.queueSvc = queueSvc

	s.tournamentSvc = tournament.NewTournamentQueueService(ctx, eventPub, botStore, gc,
		cfg.TournamentCfg.EntryFee,
		cfg.TournamentCfg.MinPlayersRequired,
		cfg.TournamentCfg.IntervalSeconds,
		cfg.TournamentCfg.BeforeStartSeconds,
		cfg.TournamentCfg.BotFillWindowSeconds,
		cfg.TournamentCfg.BotFillIntervalSeconds,
		cfg.BotRegistryFreshnessSec,
	)
	s.grpcHandlers = NewGRPCServices(s.queueSvc, s.tournamentSvc, rw)

	s.grpcServer = grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryServerInterceptor))
	proto.RegisterLobbyServiceServer(s.grpcServer, s.grpcHandlers)

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

// Start queue, tournament coordinator, gRPC timer worker, and listener.
func (s *Service) Start() error {
	log.Debugw("lobby server timer initialized")
	if err := s.queueSvc.Start(); err != nil {
		log.Errorw("queue service start failed", "err", err)
		return err
	}
	log.Debugw("queue service started")

	s.tournamentSvc.Start()
	log.Debugw("tournament tournament service started")
	s.runRoomSettlementSubscriber()
	timer.StartTimer(timer.ScopeLobby)
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
	s.tournamentSvc.Stop()
	s.queueSvc.Stop()
	timer.StopTimer(timer.ScopeLobby)
	s.grpcServer.GracefulStop()
	if s.roomConn != nil {
		_ = s.roomConn.Close()
	}
}
