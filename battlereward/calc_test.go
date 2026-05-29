package battlereward

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

const testBaseStake = 1000

func testGameArgs() *dao.GameArgs {
	return &dao.GameArgs{
		BaseStake:                testBaseStake,
		TieTokenRateBps:          80,
		TiePointRateBps:          80,
		SystemFeeRateBps:         160,
		NormalWinnerPointRateBps: 120,
		NormalLoserPointRateBps:  40,
		KOWinnerPointRateBps:     160,
	}
}

func TestComputeBattleRewardAmounts_Tie(t *testing.T) {
	gr := &dao.GameResult{
		GameResultType: proto.GameResultType_GAME_TIE,
		Multiplier:     1,
		PlayerResultInfos: []*dao.PlayerResultInfo{
			{PlayerId: 1, TemporaryAddress: "0xa", PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE},
			{PlayerId: 2, TemporaryAddress: "0xb", PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE},
		},
	}
	br := &dao.BattleRewardPVP{
		PlayerRewards: []*dao.PlayerReward{
			{PlayerId: 1},
			{PlayerId: 2},
		},
	}
	ComputeBattleRewardAmounts(gr, br, testGameArgs())
	require.Equal(t, int32(16), br.SystemFee)
	require.Equal(t, int32(-8), br.PlayerRewards[0].TokenChange)
	require.Equal(t, int32(8), br.PlayerRewards[0].PointChange)
}

func TestComputeBattleRewardAmounts_TimeoutTieZeroPayout(t *testing.T) {
	gr := &dao.GameResult{
		GameResultType: proto.GameResultType_GAME_TIE,
		Multiplier:     0,
	}
	br := &dao.BattleRewardPVP{
		PlayerRewards: []*dao.PlayerReward{{PlayerId: 1}, {PlayerId: 2}},
	}
	ComputeBattleRewardAmounts(gr, br, testGameArgs())
	require.Equal(t, int32(0), br.SystemFee)
	for _, pr := range br.PlayerRewards {
		require.Equal(t, int32(0), pr.TokenChange)
		require.Equal(t, int32(0), pr.PointChange)
	}
}

func TestComputeBattleRewardAmounts_NormalWinOfflineLoserBonus(t *testing.T) {
	gr := &dao.GameResult{
		GameResultType: proto.GameResultType_GAME_NORMAL,
		Multiplier:     1,
		PlayerResultInfos: []*dao.PlayerResultInfo{
			{PlayerId: 1, TemporaryAddress: "0xw", IsWinner: true, PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN},
			{PlayerId: 2, TemporaryAddress: "0xl", IsWinner: false, PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_OFFLINE},
		},
	}
	br := &dao.BattleRewardPVP{
		PlayerRewards: []*dao.PlayerReward{
			{PlayerId: 1},
			{PlayerId: 2},
		},
	}
	ComputeBattleRewardAmounts(gr, br, testGameArgs())
	require.Equal(t, int32(984), br.PlayerRewards[0].TokenChange)
	require.Equal(t, int32(-1000), br.PlayerRewards[1].TokenChange)
	require.Equal(t, int32(0), br.PlayerRewards[1].PointChange)
	require.Equal(t, int32(16), br.PlayerRewards[0].PointChange)
}

func TestComputeBattleRewardAmounts_KO(t *testing.T) {
	gr := &dao.GameResult{
		GameResultType: proto.GameResultType_GAME_KO,
		Multiplier:     1,
		PlayerResultInfos: []*dao.PlayerResultInfo{
			{PlayerId: 1, TemporaryAddress: "0xw", IsWinner: true, PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN},
			{PlayerId: 2, TemporaryAddress: "0xl", IsWinner: false, PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_LOSE},
		},
	}
	br := &dao.BattleRewardPVP{
		PlayerRewards: []*dao.PlayerReward{
			{PlayerId: 1},
			{PlayerId: 2},
		},
	}
	ComputeBattleRewardAmounts(gr, br, testGameArgs())
	require.Equal(t, int32(16), br.PlayerRewards[0].PointChange)
	require.Equal(t, int32(0), br.PlayerRewards[1].PointChange)
}

func TestComputeBattleRewardAmounts_UsesConfigurableRates(t *testing.T) {
	gr := &dao.GameResult{
		GameResultType: proto.GameResultType_GAME_NORMAL,
		Multiplier:     2,
		PlayerResultInfos: []*dao.PlayerResultInfo{
			{PlayerId: 1, IsWinner: true, PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_WIN},
			{PlayerId: 2, PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_LOSE},
		},
	}
	br := &dao.BattleRewardPVP{
		PlayerRewards: []*dao.PlayerReward{
			{PlayerId: 1},
			{PlayerId: 2},
		},
	}
	ga := &dao.GameArgs{
		BaseStake:                1000,
		TieTokenRateBps:          50,
		TiePointRateBps:          70,
		SystemFeeRateBps:         200,
		NormalWinnerPointRateBps: 300,
		NormalLoserPointRateBps:  100,
		KOWinnerPointRateBps:     250,
	}

	ComputeBattleRewardAmounts(gr, br, ga)

	require.Equal(t, int32(40), br.SystemFee)
	require.Equal(t, int32(1960), br.PlayerRewards[0].TokenChange)
	require.Equal(t, int32(-2000), br.PlayerRewards[1].TokenChange)
	require.Equal(t, int32(60), br.PlayerRewards[0].PointChange)
	require.Equal(t, int32(20), br.PlayerRewards[1].PointChange)
}
