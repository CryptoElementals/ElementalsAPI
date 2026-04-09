package dao

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	gameArgsValidator     *validator.Validate
	gameArgsValidatorOnce sync.Once
)

func gameArgsValidatorInstance() *validator.Validate {
	gameArgsValidatorOnce.Do(func() {
		gameArgsValidator = validator.New(validator.WithRequiredStructEnabled())
	})
	return gameArgsValidator
}

// MustValidateGameArgs panics if the row is missing or GameArgs validate tags fail.
// Call when loading the room-server template (db.LoadRoomServerGameArgs) or cloning args for a new match.
func MustValidateGameArgs(ga *GameArgs) {
	if ga == nil {
		panic("game_args: nil")
	}
	if err := gameArgsValidatorInstance().Struct(ga); err != nil {
		var ves validator.ValidationErrors
		if errors.As(err, &ves) {
			parts := make([]string, 0, len(ves))
			for _, fe := range ves {
				parts = append(parts, fmt.Sprintf("%s:%s", fe.Field(), fe.Tag()))
			}
			panic(fmt.Sprintf("game_args id=%d: %s", ga.ID, strings.Join(parts, ", ")))
		}
		panic(fmt.Sprintf("game_args id=%d: %v", ga.ID, err))
	}
}
