package dao

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEffectiveMaxRounds_PVPFromArgs(t *testing.T) {
	ga := &GameArgs{MaxNormalRounds: 3, MaxTurnsPerNormalRound: 3}
	g := &Game{GameArgs: ga}
	require.Equal(t, uint32(3), EffectiveMaxRounds(g))
}

func TestEffectiveMaxRounds_PVPIgnoresMaxExtraRoundsInArgs(t *testing.T) {
	ga := &GameArgs{MaxNormalRounds: 3, MaxExtraRounds: 99, MaxTurnsPerNormalRound: 3, MaxTurnsPerExtraRound: 1}
	g := &Game{GameArgs: ga}
	require.Equal(t, uint32(3), EffectiveMaxRounds(g))
	require.Equal(t, uint32(0), ExtraRoundsForPub(g))
	require.Equal(t, uint32(0), TurnsPerRoundForGame(g, 4))
}

func TestEffectiveMaxRounds_TournamentRegulationPlusOvertime(t *testing.T) {
	ga := &GameArgs{MaxNormalRounds: 3, MaxTurnsPerNormalRound: 3}
	g := &Game{Type: GameTypeTournament, GameArgs: ga, RegulationRounds: 3, OvertimeRoundsCap: 3}
	require.Equal(t, uint32(6), EffectiveMaxRounds(g))
	require.Equal(t, uint32(3), TurnsPerRoundForGame(g, 1))
	require.Equal(t, uint32(1), TurnsPerRoundForGame(g, 4))
}

func TestRegulationExtraRoundsForPub(t *testing.T) {
	ga := &GameArgs{MaxNormalRounds: 3, MaxExtraRounds: 2, MaxTurnsPerNormalRound: 3}
	pvp := &Game{GameArgs: ga}
	require.Equal(t, uint32(3), RegulationRoundsForPub(pvp))
	require.Equal(t, uint32(0), ExtraRoundsForPub(pvp))
	require.Equal(t, uint32(0), OvertimeTurnsPerRoundForPub(pvp))

	tour := &Game{Type: GameTypeTournament, GameArgs: ga, RegulationRounds: 3, OvertimeRoundsCap: 3}
	require.Equal(t, uint32(3), RegulationRoundsForPub(tour))
	require.Equal(t, uint32(3), ExtraRoundsForPub(tour))
	require.Equal(t, uint32(1), OvertimeTurnsPerRoundForPub(tour))
}
