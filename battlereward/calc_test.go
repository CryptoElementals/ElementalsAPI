package battlereward

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

const testBaseStake = 1000

func TestComputeBattleRewardAmounts_Tie(t *testing.T) {
	gr := &dao.GameResult{
		Multiplier:     1,
		GameResultType: proto.GameResultType_GAME_TIE,
		BattleReward: &dao.BattleReward{
			PlayerRewards: []*dao.PlayerReward{
				{PlayerId: 1, TemporaryAddress: "0xa", PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE},
				{PlayerId: 2, TemporaryAddress: "0xb", PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE},
			},
		},
	}
	ComputeBattleRewardAmounts(gr, testBaseStake)
	require.Equal(t, int32(16), gr.BattleReward.SystemFee)
	require.Equal(t, int32(-8), gr.BattleReward.PlayerRewards[0].TokenChange)
	require.Equal(t, int32(8), gr.BattleReward.PlayerRewards[0].PointChange)
}

func TestComputeBattleRewardAmounts_TimeoutTieZeroPayout(t *testing.T) {
	gr := &dao.GameResult{
		Multiplier:     0,
		GameResultType: proto.GameResultType_GAME_TIE,
		BattleReward: &dao.BattleReward{
			PlayerRewards: []*dao.PlayerReward{
				{PlayerId: 1, TemporaryAddress: "0xa"},
				{PlayerId: 2, TemporaryAddress: "0xb"},
			},
		},
	}
	ComputeBattleRewardAmounts(gr, testBaseStake)
	require.Equal(t, int32(0), gr.BattleReward.SystemFee)
	for _, pr := range gr.BattleReward.PlayerRewards {
		require.Equal(t, int32(0), pr.TokenChange)
		require.Equal(t, int32(0), pr.PointChange)
	}
}

func TestComputeBattleRewardAmounts_NormalWinOfflineLoserBonus(t *testing.T) {
	gr := &dao.GameResult{
		Multiplier:             1,
		WinnerPlayerId:         1,
		WinnerTemporaryAddress: "0xw",
		GameResultType:         proto.GameResultType_GAME_NORMAL,
		BattleReward: &dao.BattleReward{
			PlayerRewards: []*dao.PlayerReward{
				{PlayerId: 1, TemporaryAddress: "0xw", PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN},
				{PlayerId: 2, TemporaryAddress: "0xl", IsOffline: true, PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_LOSE},
			},
		},
	}
	ComputeBattleRewardAmounts(gr, testBaseStake)
	require.Equal(t, int32(984), gr.BattleReward.PlayerRewards[0].TokenChange)
	require.Equal(t, int32(-1000), gr.BattleReward.PlayerRewards[1].TokenChange)
	require.Equal(t, int32(0), gr.BattleReward.PlayerRewards[1].PointChange)
	require.Equal(t, int32(16), gr.BattleReward.PlayerRewards[0].PointChange)
}

func TestComputeBattleRewardAmounts_KO(t *testing.T) {
	gr := &dao.GameResult{
		Multiplier:             1,
		WinnerPlayerId:         1,
		WinnerTemporaryAddress: "0xw",
		GameResultType:         proto.GameResultType_GAME_KO,
		BattleReward: &dao.BattleReward{
			PlayerRewards: []*dao.PlayerReward{
				{PlayerId: 1, TemporaryAddress: "0xw", PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN},
				{PlayerId: 2, TemporaryAddress: "0xl", PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_LOSE},
			},
		},
	}
	ComputeBattleRewardAmounts(gr, testBaseStake)
	require.Equal(t, int32(16), gr.BattleReward.PlayerRewards[0].PointChange)
	require.Equal(t, int32(0), gr.BattleReward.PlayerRewards[1].PointChange)
}
