package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
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
