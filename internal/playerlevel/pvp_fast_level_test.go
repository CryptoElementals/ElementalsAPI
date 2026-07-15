package playerlevel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoostPointsAfterPVPIncreaseLevel0To3(t *testing.T) {
	got := BoostPointsAfterPVPIncrease(0, 100)
	require.Equal(t, pvpFastLevel3MinPoints, got)
}

func TestBoostPointsAfterPVPIncreaseLevel2To3(t *testing.T) {
	got := BoostPointsAfterPVPIncrease(15000, 16000)
	require.Equal(t, pvpFastLevel3MinPoints, got)
}

func TestBoostPointsAfterPVPIncreaseLevel4To6(t *testing.T) {
	got := BoostPointsAfterPVPIncrease(35000, 36000)
	require.Equal(t, pvpFastLevel6MinPoints, got)
}

func TestBoostPointsAfterPVPIncreaseLevel7To9(t *testing.T) {
	got := BoostPointsAfterPVPIncrease(175000, 176000)
	require.Equal(t, pvpFastLevel9MinPoints, got)
}

func TestBoostPointsAfterPVPIncreaseNoChangeWhenNotIncreased(t *testing.T) {
	got := BoostPointsAfterPVPIncrease(15000, 15000)
	require.Equal(t, 15000, got)
	got = BoostPointsAfterPVPIncrease(15000, 14000)
	require.Equal(t, 14000, got)
}

func TestBoostPointsAfterPVPIncreaseNoChangeAtLevel9Plus(t *testing.T) {
	got := BoostPointsAfterPVPIncrease(260000, 261000)
	require.Equal(t, 261000, got)
}

func TestBoostPointsAfterPVPIncreaseKeepsHigherNaturalGain(t *testing.T) {
	got := BoostPointsAfterPVPIncrease(15000, 30000)
	require.Equal(t, 30000, got)
}

func TestIsPvpFastLevelPlayer(t *testing.T) {
	SetPvpFastLevelPlayerIDs([]int64{1001, 1002})
	require.True(t, IsPvpFastLevelPlayer(1001))
	require.False(t, IsPvpFastLevelPlayer(9999))
	SetPvpFastLevelPlayerIDs(nil)
	require.False(t, IsPvpFastLevelPlayer(1001))
}
