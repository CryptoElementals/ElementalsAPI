package dao

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validGameArgs() *GameArgs {
	return &GameArgs{
		BaseModel:                             BaseModel{ID: 1},
		MaxNormalRounds:                       3,
		MaxExtraRounds:                        0,
		MaxTurnsPerNormalRound:                3,
		MaxTurnsPerExtraRound:                 0,
		InitialHP:                             3000,
		BaseStake:                             1000,
		ConfirmationTimeout:                   10,
		CommitmentSubmissionTimeout:           20,
		CardSubmissionTimeout:                 20,
		GameContinueTimeout:                   10,
		ConfirmationTimeoutRedundancy:         1,
		CommitmentSubmissionTimeoutRedundancy: 1,
		CardSubmissionTimeoutRedundancy:       1,
		GameContinueTimeoutRedundancy:         1,
	}
}

func TestMustValidateGameArgs_ok(t *testing.T) {
	require.NotPanics(t, func() { MustValidateGameArgs(validGameArgs()) })
}

func TestMustValidateGameArgs_panics(t *testing.T) {
	ga := validGameArgs()
	ga.MaxNormalRounds = 0
	require.Panics(t, func() { MustValidateGameArgs(ga) })
}

func TestMustValidateGameArgs_nil(t *testing.T) {
	require.Panics(t, func() { MustValidateGameArgs(nil) })
}
