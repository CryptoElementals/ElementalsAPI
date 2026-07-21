package invite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeriveInviterRewardStatusLocked(t *testing.T) {
	status, next := DeriveInviterRewardStatus(2, nil)
	require.Equal(t, InviterRewardLocked, status)
	require.Equal(t, 3, next)
}

func TestDeriveInviterRewardStatusClaimable(t *testing.T) {
	status, next := DeriveInviterRewardStatus(5, []MilestoneRewardState{
		{MilestoneLevel: 3, Claimed: false},
	})
	require.Equal(t, InviterRewardClaimable, status)
	require.Equal(t, 3, next)

	status, next = DeriveInviterRewardStatus(10, []MilestoneRewardState{
		{MilestoneLevel: 3, Claimed: true},
		{MilestoneLevel: 6, Claimed: false},
		{MilestoneLevel: 9, Claimed: false},
	})
	require.Equal(t, InviterRewardClaimable, status)
	require.Equal(t, 6, next)
}

func TestDeriveInviterRewardStatusWaiting(t *testing.T) {
	status, next := DeriveInviterRewardStatus(5, []MilestoneRewardState{
		{MilestoneLevel: 3, Claimed: true},
	})
	require.Equal(t, InviterRewardWaiting, status)
	require.Equal(t, 6, next)
}

func TestDeriveInviterRewardStatusCompleted(t *testing.T) {
	status, next := DeriveInviterRewardStatus(12, []MilestoneRewardState{
		{MilestoneLevel: 3, Claimed: true},
		{MilestoneLevel: 6, Claimed: true},
		{MilestoneLevel: 9, Claimed: true},
		{MilestoneLevel: 12, Claimed: true},
	})
	require.Equal(t, InviterRewardCompleted, status)
	require.Equal(t, 0, next)
}

func TestDeriveInviterRewardItemStatus(t *testing.T) {
	require.Equal(t, InviterRewardItemLocked, DeriveInviterRewardItemStatus(false, false))
	require.Equal(t, InviterRewardItemClaimable, DeriveInviterRewardItemStatus(true, false))
	require.Equal(t, InviterRewardItemClaimed, DeriveInviterRewardItemStatus(true, true))
	// claimed without a row should still report claimed
	require.Equal(t, InviterRewardItemClaimed, DeriveInviterRewardItemStatus(false, true))
}
