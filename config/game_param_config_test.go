package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGameParams_EnableDailyReward_DefaultFalse(t *testing.T) {
	InitializeGameParams(&GameParamConfig{})
	require.False(t, GameParams.EnableDailyReward)
}

func TestGameParams_EnableDailyReward_ExplicitTrue(t *testing.T) {
	InitializeGameParams(&GameParamConfig{EnableDailyReward: true})
	require.True(t, GameParams.EnableDailyReward)
}
