package db

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

// CreateBot creates a user_profile and user_token for a bot with the given playerID, name, avatar/background URLs, token amount and points.
// name must be unique (e.g. "bot_1" or "Bot_12345").
func CreateBot(playerID int64, name string, avatarURL string, backgroundURL string, tokenAmount int32, points int32) (*dao.UserProfile, *dao.UserToken, error) {
	var profile dao.UserProfile
	var token dao.UserToken
	err := Get().Transaction(func(tx *gorm.DB) error {
		profile = dao.UserProfile{
			PlayerID:      playerID,
			Name:          name,
			AvatarURL:     avatarURL,
			BackgroundURL: backgroundURL,
		}
		if err := tx.Create(&profile).Error; err != nil {
			return err
		}
		token = dao.UserToken{
			PlayerId:    profile.PlayerID,
			Points:      points,
			TokenAmount: tokenAmount,
		}
		if err := tx.Create(&token).Error; err != nil {
			return fmt.Errorf("create user_token for bot: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &profile, &token, nil
}
