package invite

import (
	"context"
	"net"
	"testing"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	gorm_logger "gorm.io/gorm/logger"
)

type testLobbyService struct {
	proto.UnimplementedLobbyServiceServer
}

func (s *testLobbyService) CreditUserPoints(ctx context.Context, req *proto.CreditUserPointsRequest) (*proto.GetPlayerTokenResponse, error) {
	userToken, err := db.CreditUserPointsAmount(req.GetPlayerID(), req.GetDelta())
	if err != nil {
		return nil, err
	}
	return &proto.GetPlayerTokenResponse{
		Id:     userToken.PlayerId,
		Points: uint64(userToken.Points),
		Tokens: uint64(userToken.TokenAmount),
	}, nil
}

func (s *testLobbyService) GetPlayerToken(ctx context.Context, req *proto.GetPlayerTokenRequest) (*proto.GetPlayerTokenResponse, error) {
	token, err := db.GetPlayerToken(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &proto.GetPlayerTokenResponse{
		Id:     token.PlayerId,
		Points: uint64(token.Points),
		Tokens: uint64(token.TokenAmount),
	}, nil
}

// SetUpTestDB initializes an in-memory database for invite-related tests.
func SetUpTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, db.Init(&db.Config{Development: true}))
	db.Get().Logger = db.Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, db.MigrateMemDb())
}

// SetUpDualLobbyClients wires trial/normal lobby gRPC clients for invite API tests.
func SetUpDualLobbyClients(t *testing.T) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	proto.RegisterLobbyServiceServer(srv, &testLobbyService{})
	go func() { _ = srv.Serve(lis) }()
	dial := func() *grpc.ClientConn {
		conn, err := grpc.DialContext(context.Background(), "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		return conn
	}
	trialConn := dial()
	normalConn := dial()
	client.SetLobbyClientForTest(config.ServerTypeTrial, proto.NewLobbyServiceClient(trialConn))
	client.SetLobbyClientForTest(config.ServerTypeNormal, proto.NewLobbyServiceClient(normalConn))
	config.GConf = config.ApiServerConfig{
		InviteeInitialPoints: 50,
		MaxInviteesPerCode:   3,
		EnvironmentConfigs: []config.EnvironmentConfig{
			{Name: config.ServerTypeTrial},
			{Name: config.ServerTypeNormal},
		},
	}
	t.Cleanup(func() {
		client.SetLobbyClientForTest(config.ServerTypeTrial, nil)
		client.SetLobbyClientForTest(config.ServerTypeNormal, nil)
		_ = trialConn.Close()
		_ = normalConn.Close()
		srv.Stop()
		_ = lis.Close()
	})
}
