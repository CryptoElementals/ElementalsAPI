package db

import dao "github.com/CryptoElementals/common/models"

func LoadGameArgsByID(gameArgsID uint) (*dao.GameArgs, error) {
	var gameArgs dao.GameArgs
	if err := Get().First(&gameArgs, gameArgsID).Error; err != nil {
		return nil, err
	}
	return &gameArgs, nil
}
