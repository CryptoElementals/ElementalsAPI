package playerlevel

// Level thresholds — kept in sync with legacy get_user_profile.calculateLevel.
var levelThresholds = []int{
	0, 5000, 10000, 20500, 35000, 54500, 81000, 120000, 175000, 260000,
	410000, 650000, 1050000, 1700000, 2800000, 4600000, 7500000, 12300000,
	20500000, 35000000, 60000000,
}

// CalculateLevel returns the player level for the given points.
func CalculateLevel(points int) int {
	level, _, _ := CalculateLevelDetail(points)
	return level
}

// CalculateLevelDetail returns level, current level threshold, and next level threshold.
func CalculateLevelDetail(points int) (level int, currentLevelPoints int, nextLevelPoints int) {
	if points == 0 {
		return 0, 0, levelThresholds[1]
	}
	for i, threshold := range levelThresholds {
		if points < threshold {
			level = i - 1
			if level >= 0 {
				currentLevelPoints = levelThresholds[level]
			}
			if i < len(levelThresholds) {
				nextLevelPoints = levelThresholds[i]
			} else {
				nextLevelPoints = levelThresholds[len(levelThresholds)-1]
			}
			return
		}
	}
	level = len(levelThresholds) - 1
	currentLevelPoints = levelThresholds[level]
	nextLevelPoints = levelThresholds[len(levelThresholds)-1]
	return
}
