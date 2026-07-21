package lobbyserver

import (
	"context"
	"testing"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
	gorm_logger "gorm.io/gorm/logger"
)

func setupLobbyHandlerTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, db.Init(&db.Config{Development: true}))
	db.Get().Logger = db.Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, db.MigrateMemDb())
}

func TestCreditUserPointsHandler(t *testing.T) {
	setupLobbyHandlerTestDB(t)
	require.NoError(t, db.Get().Create(&dao.UserToken{PlayerId: 9201, Points: 10, TokenAmount: 0}).Error)

	svc := &GRPCServices{}
	resp, err := svc.CreditUserPoints(context.Background(), &proto.CreditUserPointsRequest{
		PlayerID: 9201,
		Delta:    40,
		Reason:   "test",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(50), resp.GetPoints())
}
