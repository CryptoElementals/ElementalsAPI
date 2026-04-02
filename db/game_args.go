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
