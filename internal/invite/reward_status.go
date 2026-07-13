package invite

// InviterRewardStatus describes inviter milestone reward progress for one invitee.
type InviterRewardStatus string

const (
	// InviterRewardLocked: invitee below Lv.3; inviter has no milestone rewards yet.
	InviterRewardLocked InviterRewardStatus = "locked"
	// InviterRewardClaimable: at least one milestone reward is ready to claim.
	InviterRewardClaimable InviterRewardStatus = "claimable"
	// InviterRewardWaiting: current milestones are claimed; invitee must level up for the next reward.
	InviterRewardWaiting InviterRewardStatus = "waiting"
	// InviterRewardCompleted: all milestone rewards (Lv.3/6/9/12) have been claimed.
	InviterRewardCompleted InviterRewardStatus = "completed"
)

// MilestoneRewardState is one inviter milestone row for status derivation.
type MilestoneRewardState struct {
	MilestoneLevel int
	Claimed        bool
}

// DeriveInviterRewardStatus returns an explicit status and the next milestone level
// the invitee must reach (0 when all milestones are done).
func DeriveInviterRewardStatus(inviteeLevel int, rewards []MilestoneRewardState) (InviterRewardStatus, int) {
	claimed := make(map[int]bool, len(DefaultInviterMilestones))
	lowestUnclaimed := 0
	hasUnclaimed := false

	for _, reward := range rewards {
		if reward.Claimed {
			claimed[reward.MilestoneLevel] = true
			continue
		}
		hasUnclaimed = true
		if lowestUnclaimed == 0 || reward.MilestoneLevel < lowestUnclaimed {
			lowestUnclaimed = reward.MilestoneLevel
		}
	}

	if hasUnclaimed {
		return InviterRewardClaimable, lowestUnclaimed
	}

	allClaimed := true
	for _, milestone := range DefaultInviterMilestones {
		if !claimed[milestone.Level] {
			allClaimed = false
			break
		}
	}
	if allClaimed {
		return InviterRewardCompleted, 0
	}

	if inviteeLevel < DefaultInviterMilestones[0].Level && len(rewards) == 0 {
		return InviterRewardLocked, DefaultInviterMilestones[0].Level
	}

	return InviterRewardWaiting, nextTargetMilestoneLevel(inviteeLevel, claimed)
}

func nextTargetMilestoneLevel(inviteeLevel int, claimed map[int]bool) int {
	for _, milestone := range DefaultInviterMilestones {
		if inviteeLevel < milestone.Level || !claimed[milestone.Level] {
			return milestone.Level
		}
	}
	return 0
}
