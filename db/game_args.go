package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
)

const (
	defaultTieTokenRateBps          = 80
	defaultTiePointRateBps          = 80
	defaultSystemFeeRateBps         = 160
	defaultNormalWinnerPointRateBps = 120
	defaultNormalLoserPointRateBps  = 40
	defaultKOWinnerPointRateBps     = 160
)

// LoadRoomServerGameArgs loads the template row used for new matches by primary key.
// Deployment must set game-args-id to a non-zero existing row; room server fatals if the load fails.
// Returns a heap copy safe to keep in memory.
func LoadRoomServerGameArgs(templateID uint) (*dao.GameArgs, error) {
	var row dao.GameArgs
	if err := Get().First(&row, templateID).Error; err != nil {
		return nil, fmt.Errorf("game_args template (id=%d): %w", templateID, err)
	}
	out := row
	dao.MustValidateGameArgs(&out)
	return &out, nil
}

// LoadAllGameArgs loads all game_args rows and returns heap copies keyed by id.
func LoadAllGameArgs() (map[uint]*dao.GameArgs, error) {
	var rows []dao.GameArgs
	if err := Get().Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load all game_args: %w", err)
	}
	out := make(map[uint]*dao.GameArgs, len(rows))
	for i := range rows {
		rowCopy := rows[i]
		dao.MustValidateGameArgs(&rowCopy)
		out[rowCopy.ID] = &rowCopy
	}
	return out, nil
}

// BackfillGameArgsRewardRates sets defaults for nullable reward-rate columns after schema migration.
// This is safe to run repeatedly and only updates rows where any new field is NULL.
func BackfillGameArgsRewardRates() error {
	return Get().Exec(`
UPDATE game_args
SET
	tie_token_rate_bps = COALESCE(tie_token_rate_bps, ?),
	tie_point_rate_bps = COALESCE(tie_point_rate_bps, ?),
	system_fee_rate_bps = COALESCE(system_fee_rate_bps, ?),
	normal_winner_point_rate_bps = COALESCE(normal_winner_point_rate_bps, ?),
	normal_loser_point_rate_bps = COALESCE(normal_loser_point_rate_bps, ?),
	ko_winner_point_rate_bps = COALESCE(ko_winner_point_rate_bps, ?)
WHERE
	tie_token_rate_bps IS NULL
	OR tie_point_rate_bps IS NULL
	OR system_fee_rate_bps IS NULL
	OR normal_winner_point_rate_bps IS NULL
	OR normal_loser_point_rate_bps IS NULL
	OR ko_winner_point_rate_bps IS NULL
`,
		defaultTieTokenRateBps,
		defaultTiePointRateBps,
		defaultSystemFeeRateBps,
		defaultNormalWinnerPointRateBps,
		defaultNormalLoserPointRateBps,
		defaultKOWinnerPointRateBps,
	).Error
}
