package dao

// TurnsPerRoundFromArgs returns MaxTurnsPerRound from persisted match parameters.
func TurnsPerRoundFromArgs(ga *GameArgs) uint32 {
	return uint32(ga.MaxTurnsPerRound)
}

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
