package api

import (
	"context"
	"net"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	cmnErrors "github.com/CryptoElementals/common/errors"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	gorm_logger "gorm.io/gorm/logger"
)

func setupTestDBForNewUserRewardAPI(t *testing.T) {
	t.Helper()
	require.NoError(t, db.Init(&db.Config{Development: true}))
	db.Get().Logger = db.Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, db.MigrateMemDb())
	setupTestLobbyClientForRewards(t)
}

func setupGinContext() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	return c
}

type testLobbyRewardService struct {
	proto.UnimplementedLobbyServiceServer
}

func (s *testLobbyRewardService) CreditUserTokens(ctx context.Context, req *proto.CreditUserTokensRequest) (*proto.GetPlayerTokenResponse, error) {
	userToken, err := db.CreditUserTokenAmount(req.GetPlayerID(), req.GetDelta())
	if err != nil {
		return nil, err
	}
	return &proto.GetPlayerTokenResponse{
		Id:           userToken.PlayerId,
		Tokens:       uint64(userToken.TokenAmount),
		Points:       uint64(userToken.Points),
		LockedTokens: 0,
	}, nil
}

func setupTestLobbyClientForRewards(t *testing.T) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	proto.RegisterLobbyServiceServer(srv, &testLobbyRewardService{})
	go func() {
		_ = srv.Serve(lis)
	}()
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	client.SetGlobalLobbyClientForTest(proto.NewLobbyServiceClient(conn))
	t.Cleanup(func() {
		client.SetGlobalLobbyClientForTest(nil)
		_ = conn.Close()
		srv.Stop()
		_ = lis.Close()
	})
}

func TestCollectNewUserReward_OnlyOnce(t *testing.T) {
	setupTestDBForNewUserRewardAPI(t)
	config.InitializeGameParams(&config.GameParamConfig{NewUserRewardTokens: 1234, EnableNewUserReward: true})

	profile := &dao.UserProfile{
		PlayerID: 4001,
		Address:  "0xapi_new_reward_1",
		Name:     "api_new_reward_1",
	}
	require.NoError(t, db.Get().Create(profile).Error)
	require.NoError(t, db.Get().Create(&dao.UserToken{PlayerId: profile.PlayerID, Points: 0, TokenAmount: 100}).Error)

	playerID := strconv.FormatInt(profile.PlayerID, 10)
	params := map[string]interface{}{
		"Action":      COLLECT_NEW_USER_REWARD_LABEL,
		"RequestUUID": "test-request-1",
		"PlayerID":    playerID,
	}
	taskIntf, err := NewCollectNewUserRewardTask(&params)
	require.NoError(t, err)

	resp, err := taskIntf.Run(setupGinContext())
	require.NoError(t, err)
	typedResp, ok := resp.(*CollectNewUserRewardResponse)
	require.True(t, ok)
	require.Equal(t, int32(1234), typedResp.RewardAmount)

	var token dao.UserToken
	require.NoError(t, db.Get().Where("player_id = ?", profile.PlayerID).First(&token).Error)
	require.Equal(t, int32(1334), token.TokenAmount)

	_, err = taskIntf.Run(setupGinContext())
	require.Error(t, err)
	customErr, ok := err.(cmnErrors.Error)
	require.True(t, ok)
	require.Equal(t, int(cmnErrors.ActionError().Code()), int(customErr.Code()))
	require.Contains(t, customErr.String(), "already collected")

	require.NoError(t, db.Get().Where("player_id = ?", profile.PlayerID).First(&token).Error)
	require.Equal(t, int32(1334), token.TokenAmount)
}

func TestHasCollectedNewUserReward_StatusChangesAfterClaim(t *testing.T) {
	setupTestDBForNewUserRewardAPI(t)
	config.InitializeGameParams(&config.GameParamConfig{NewUserRewardTokens: 200, EnableNewUserReward: true})

	profile := &dao.UserProfile{
		PlayerID: 4002,
		Address:  "0xapi_new_reward_2",
		Name:     "api_new_reward_2",
	}
	require.NoError(t, db.Get().Create(profile).Error)
	require.NoError(t, db.Get().Create(&dao.UserToken{PlayerId: profile.PlayerID, Points: 0, TokenAmount: 0}).Error)

	playerID := strconv.FormatInt(profile.PlayerID, 10)
	checkParams := map[string]interface{}{
		"Action":      HAS_COLLECTED_NEW_USER_REWARD_LABEL,
		"RequestUUID": "test-request-2",
		"PlayerID":    playerID,
	}
	checkTaskIntf, err := NewHasCollectedNewUserRewardTask(&checkParams)
	require.NoError(t, err)

	resp, err := checkTaskIntf.Run(setupGinContext())
	require.NoError(t, err)
	checkResp, ok := resp.(*HasCollectedNewUserRewardResponse)
	require.True(t, ok)
	require.False(t, checkResp.Collected)

	collectParams := map[string]interface{}{
		"Action":      COLLECT_NEW_USER_REWARD_LABEL,
		"RequestUUID": "test-request-3",
		"PlayerID":    playerID,
	}
	collectTaskIntf, err := NewCollectNewUserRewardTask(&collectParams)
	require.NoError(t, err)
	_, err = collectTaskIntf.Run(setupGinContext())
	require.NoError(t, err)

	resp, err = checkTaskIntf.Run(setupGinContext())
	require.NoError(t, err)
	checkResp, ok = resp.(*HasCollectedNewUserRewardResponse)
	require.True(t, ok)
	require.True(t, checkResp.Collected)
}

func TestNewUserReward_DisabledByConfig(t *testing.T) {
	setupTestDBForNewUserRewardAPI(t)
	config.InitializeGameParams(&config.GameParamConfig{NewUserRewardTokens: 999, EnableNewUserReward: false})

	profile := &dao.UserProfile{
		PlayerID: 4003,
		Address:  "0xapi_new_reward_disabled",
		Name:     "api_new_reward_disabled",
	}
	require.NoError(t, db.Get().Create(profile).Error)
	require.NoError(t, db.Get().Create(&dao.UserToken{PlayerId: profile.PlayerID, Points: 0, TokenAmount: 50}).Error)
	playerID := strconv.FormatInt(profile.PlayerID, 10)

	checkParams := map[string]interface{}{
		"Action":      HAS_COLLECTED_NEW_USER_REWARD_LABEL,
		"RequestUUID": "test-request-disabled-has",
		"PlayerID":    playerID,
	}
	checkTaskIntf, err := NewHasCollectedNewUserRewardTask(&checkParams)
	require.NoError(t, err)
	resp, err := checkTaskIntf.Run(setupGinContext())
	require.NoError(t, err)
	checkResp, ok := resp.(*HasCollectedNewUserRewardResponse)
	require.True(t, ok)
	require.True(t, checkResp.Collected)

	collectParams := map[string]interface{}{
		"Action":      COLLECT_NEW_USER_REWARD_LABEL,
		"RequestUUID": "test-request-disabled-collect",
		"PlayerID":    playerID,
	}
	collectTaskIntf, err := NewCollectNewUserRewardTask(&collectParams)
	require.NoError(t, err)
	_, err = collectTaskIntf.Run(setupGinContext())
	require.Error(t, err)
	customErr, ok := err.(cmnErrors.Error)
	require.True(t, ok)
	require.Contains(t, customErr.String(), "not enabled")

	var token dao.UserToken
	require.NoError(t, db.Get().Where("player_id = ?", profile.PlayerID).First(&token).Error)
	require.Equal(t, int32(50), token.TokenAmount)
}
