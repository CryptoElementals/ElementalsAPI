package dao

// MaxRoundNumberFromTurns returns the highest round_number present in persisted turns.
func MaxRoundNumberFromTurns(turns []*Turn) uint32 {
	var m uint32
	for _, t := range turns {
		if t == nil {
			continue
		}
		if t.RoundNumber > m {
			m = t.RoundNumber
		}
	}
	return m
}

const (
	// GameTypeTournament matches proto.GameType_TOURNAMENT.
	GameTypeTournament uint = 2

	// TournamentMaxOvertimeRounds is the cap on tie-breaker rounds (one turn each) after regulation.
	TournamentMaxOvertimeRounds uint32 = 3
)

func regulationRoundsCount(g *Game) uint32 {
	if g == nil || g.GameArgs == nil {
		return 0
	}
	if g.Type == GameTypeTournament && g.RegulationRounds > 0 {
		return g.RegulationRounds
	}
	return uint32(g.GameArgs.MaxNormalRounds)
}

func tournamentOvertimeRoundsCount(g *Game) uint32 {
	if g == nil || g.Type != GameTypeTournament {
		return 0
	}
	if g.OvertimeRoundsCap > 0 {
		return g.OvertimeRoundsCap
	}
	return TournamentMaxOvertimeRounds
}

// normalTurnsPerRound is regulation-phase turns per round; 0 if game or args missing.
func normalTurnsPerRound(g *Game) uint32 {
	if g == nil || g.GameArgs == nil {
		return 0
	}
	return uint32(g.GameArgs.MaxTurnsPerNormalRound)
}

// EffectiveMaxRounds is total scheduled rounds: regulation for all modes, plus tournament overtime only.
func EffectiveMaxRounds(g *Game) uint32 {
	return regulationRoundsCount(g) + tournamentOvertimeRoundsCount(g)
}

// TurnsPerRoundForGame returns card turns in roundNumber (1-based).
func TurnsPerRoundForGame(g *Game, roundNumber uint32) uint32 {
	if roundNumber < 1 {
		return 0
	}
	reg := regulationRoundsCount(g)
	if roundNumber <= reg {
		return normalTurnsPerRound(g)
	}
	if g == nil || g.Type != GameTypeTournament {
		return 0
	}
	return 1
}

// RegulationRoundsForPub is regulation round count exposed to clients (GameReady, etc.).
func RegulationRoundsForPub(g *Game) uint32 {
	return regulationRoundsCount(g)
}

// ExtraRoundsForPub is tournament overtime round count for clients; zero for PVP.
func ExtraRoundsForPub(g *Game) uint32 {
	return tournamentOvertimeRoundsCount(g)
}

// RegulationTurnsPerRoundForPub is turns per regulation round for clients.
func RegulationTurnsPerRoundForPub(g *Game) uint32 {
	return normalTurnsPerRound(g)
}

// OvertimeTurnsPerRoundForPub is 1 for tournament overtime rounds, 0 for PVP.
func OvertimeTurnsPerRoundForPub(g *Game) uint32 {
	if g == nil || g.Type != GameTypeTournament {
		return 0
	}
	return 1
}
