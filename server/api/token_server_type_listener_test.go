package api

import (
	"testing"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
	gorm_logger "gorm.io/gorm/logger"
)

func setupTestDBForTokenServerTypeListener(t *testing.T) {
	t.Helper()
	require.NoError(t, db.Init(&db.Config{Development: true}))
	db.Get().Logger = db.Get().Logger.LogMode(gorm_logger.Error)
	require.NoError(t, db.MigrateMemDb())
}

func TestHandleTokenServerTypePromotion(t *testing.T) {
	setupTestDBForTokenServerTypeListener(t)

	profile := &dao.UserProfile{
		PlayerID:   5001,
		Address:    "0xtokenlistener1",
		Name:       "token_listener_trial",
		ServerType: dao.ServerTypeTrial,
	}
	require.NoError(t, db.Get().Create(profile).Error)

	handleTokenServerTypePromotion(&proto.Message{
		Event: &proto.Event{
			Type: proto.EventType_TYPE_TOKEN_UPDATED,
			Event: &proto.Event_TokenUpdated{
				TokenUpdated: &proto.TokenUpdated{
					PlayerId:       5001,
					Source:         tokenSourceChainDeposit,
					DepositAddress: "0xDepositorWallet",
				},
			},
		},
	})

	updated, err := db.GetUserProfileByPlayerIDInt(5001)
	require.NoError(t, err)
	require.Equal(t, dao.ServerTypeNormal, updated.ServerType)
	require.Equal(t, "0xdepositorwallet", updated.Address)

	handleTokenServerTypePromotion(nil)
	handleTokenServerTypePromotion(&proto.Message{
		Topic: pubsub.TopicToken,
		Event: &proto.Event{Type: proto.EventType_TYPE_TOKEN_UPDATED},
	})
}
