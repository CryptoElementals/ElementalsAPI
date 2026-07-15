package playerlevel

// PVP fast-level milestone targets (level 3 / 6 / 9 minimum points).
const (
	pvpFastLevel3MinPoints = 20500
	pvpFastLevel6MinPoints = 81000
	pvpFastLevel9MinPoints = 260000
)

var pvpFastLevelPlayerIDs map[int64]struct{}

// SetPvpFastLevelPlayerIDs registers player IDs for PVP fast-level boost (lobby startup).
func SetPvpFastLevelPlayerIDs(ids []int64) {
	if len(ids) == 0 {
		pvpFastLevelPlayerIDs = nil
		return
	}
	pvpFastLevelPlayerIDs = make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		pvpFastLevelPlayerIDs[id] = struct{}{}
	}
}

// IsPvpFastLevelPlayer reports whether playerID is in the PVP fast-level whitelist.
func IsPvpFastLevelPlayer(playerID int64) bool {
	if len(pvpFastLevelPlayerIDs) == 0 {
		return false
	}
	_, ok := pvpFastLevelPlayerIDs[playerID]
	return ok
}

// BoostPointsAfterPVPIncrease bumps points to the next milestone floor (3/6/9) when
// points increased and the pre-settlement level is below 9.
func BoostPointsAfterPVPIncrease(pointsBefore, pointsAfter int) int {
	if pointsAfter <= pointsBefore {
		return pointsAfter
	}
	level := CalculateLevel(pointsBefore)
	if level >= 9 {
		return pointsAfter
	}
	minTarget := pvpFastLevelMinPointsForLevel(level)
	if pointsAfter < minTarget {
		return minTarget
	}
	return pointsAfter
}

func pvpFastLevelMinPointsForLevel(level int) int {
	switch {
	case level < 3:
		return pvpFastLevel3MinPoints
	case level < 6:
		return pvpFastLevel6MinPoints
	default:
		return pvpFastLevel9MinPoints
	}
}
