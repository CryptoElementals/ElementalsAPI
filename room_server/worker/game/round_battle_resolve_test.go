package game

import (
	"testing"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

func TestRound_isExtraRound(t *testing.T) {
	ga := &dao.GameArgs{MaxNormalRounds: 3, MaxTurnsPerNormalRound: 3}
	pvp := &round{game: &dao.Game{GameArgs: ga, Type: uint(proto.GameType_PVP)}, roundNumber: 4}
	require.False(t, pvp.isExtraRound())

	tourReg := &round{game: &dao.Game{GameArgs: ga, Type: dao.GameTypeTournament, RegulationRounds: 3, OvertimeRoundsCap: 3}, roundNumber: 3}
	require.False(t, tourReg.isExtraRound())

	tourOT := &round{game: &dao.Game{GameArgs: ga, Type: dao.GameTypeTournament, RegulationRounds: 3, OvertimeRoundsCap: 3}, roundNumber: 4}
	require.True(t, tourOT.isExtraRound())
}

func TestRound_isNextRoundExtra(t *testing.T) {
	ga := &dao.GameArgs{MaxNormalRounds: 3, MaxTurnsPerNormalRound: 3}
	pvp := &round{game: &dao.Game{GameArgs: ga, Type: uint(proto.GameType_PVP)}, roundNumber: 2}
	require.False(t, pvp.isNextRoundExtra())

	gTour := &dao.Game{GameArgs: ga, Type: dao.GameTypeTournament, RegulationRounds: 3, OvertimeRoundsCap: 3}
	require.False(t, (&round{game: gTour, roundNumber: 1}).isNextRoundExtra())
	require.True(t, (&round{game: gTour, roundNumber: 3}).isNextRoundExtra())
	require.True(t, (&round{game: gTour, roundNumber: 4}).isNextRoundExtra())
}

// Unequal HP after the last regulation turn should end the game; do not require full OT schedule.
func TestCheckGameOverByHP_regulationHPSpreadEndsBeforeOvertime(t *testing.T) {
	ga := &dao.GameArgs{MaxNormalRounds: 3, MaxTurnsPerNormalRound: 3}
	g := &dao.Game{
		Type:              dao.GameTypeTournament,
		GameArgs:          ga,
		RegulationRounds:  3,
		OvertimeRoundsCap: 3,
	}
	r := &round{game: g, roundNumber: 3, turnNumber: 3}
	require.False(t, r.isGameEndsByRoundAndTurn())
	require.True(t, r.isGameEndsByRegulationRoundAndTurn())

	states := []*gameEndState{
		{HP: 1100, PlayerId: 1, TemporaryAddress: "a", Status: playerStatusOnline},
		{HP: 1000, PlayerId: 2, TemporaryAddress: "b", Status: playerStatusOnline},
	}
	ok, grType, winnerID, winnerTemp, _ := r.checkGameOverByHP(states, false, true)
	require.True(t, ok)
	require.Equal(t, gameResultNormal, grType)
	require.Equal(t, int64(1), winnerID)
	require.Equal(t, "a", winnerTemp)
}

// Overtime used to keep playing when HP differed before the final scheduled round; tie-break should end as soon as HP splits.
func TestCheckGameOverByHP_overtimeHPSpreadEndsImmediately(t *testing.T) {
	ga := &dao.GameArgs{MaxNormalRounds: 3, MaxTurnsPerNormalRound: 3}
	g := &dao.Game{
		Type:              dao.GameTypeTournament,
		GameArgs:          ga,
		RegulationRounds:  3,
		OvertimeRoundsCap: 3,
	}
	// First OT round (4), single turn — not last overall round.
	r := &round{game: g, roundNumber: 4, turnNumber: 1}
	require.True(t, r.isExtraRound())
	require.False(t, r.isGameEndsByRoundAndTurn())
	require.True(t, r.isGameEndsByRegulationRoundAndTurn())

	states := []*gameEndState{
		{HP: 1100, PlayerId: 1, TemporaryAddress: "a", Status: playerStatusOnline},
		{HP: 1000, PlayerId: 2, TemporaryAddress: "b", Status: playerStatusOnline},
	}
	ok, grType, winnerID, winnerTemp, _ := r.checkGameOverByHP(states, false, true)
	require.True(t, ok)
	require.Equal(t, gameResultNormal, grType)
	require.Equal(t, int64(1), winnerID)
	require.Equal(t, "a", winnerTemp)
}
