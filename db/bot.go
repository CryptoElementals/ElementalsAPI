package db

import (
	"errors"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

func CreateOrCheckBot(walletAddress string, forceBot ...bool) error {
	shouldOverwriteBot := len(forceBot) != 0 && forceBot[0]
	return Get().Transaction(func(tx *gorm.DB) error {
		var userProfile dao.UserProfile
		err := tx.Where("address = ?", walletAddress).First(&userProfile).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				userProfile := &dao.UserProfile{
					Address: walletAddress,
					Name:    "Bot_" + walletAddress, // 默认用户名为 Bot_ + 钱包地址
					IsBot:   true,                   // 标记为机器人
				}
				userToken := dao.UserToken{
					WalletAddress: walletAddress,
					TokenAmount:   100000,
				}
				err = tx.Save(&userToken).Error
				if err != nil {
					return err
				}
				err = tx.Save(userProfile).Error
				if err != nil {
					return err
				}
				return nil
			}
			return err
		}
		if !userProfile.IsBot {
			if shouldOverwriteBot {
				userProfile.IsBot = true
				err = tx.Save(userProfile).Error
				if err != nil {
					return err
				}
			} else {
				return errors.New("user profile is not a bot")
			}
		}
		ut := &dao.UserToken{}
		err = tx.Where("wallet_address = ?", walletAddress).First(ut).Error
		if err != nil {
			return err
		}
		if ut.TokenAmount <= 100000 {
			ut.TokenAmount = 100000
			err = tx.Save(ut).Error
			if err != nil {
				return err
			}
		}
		return nil
	})

}
