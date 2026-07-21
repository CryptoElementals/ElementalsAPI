package invite

// Milestone defines an inviter reward tier when invitee reaches a level.
type Milestone struct {
	Level int
	Point int32
}

// DefaultInviterMilestones is the default inviter milestone table.
var DefaultInviterMilestones = []Milestone{
	{Level: 3, Point: 25},
	{Level: 6, Point: 100},
	{Level: 9, Point: 400},
	{Level: 12, Point: 1300},
}

// PointForLevel returns the reward points for a milestone level, or 0 if unknown.
func PointForLevel(level int) int32 {
	for _, m := range DefaultInviterMilestones {
		if m.Level == level {
			return m.Point
		}
	}
	return 0
}
