package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

const maxLockTime = time.Minute
const maxPlayerPerAddress = 3

func SaveUserToken(tokens ...dao.UserToken) error {
	return Get().Save(&tokens).Error
}

func LockUserToken(ctx context.Context, address string, tempAddress string, tokenAmount int32) (err error) {
	return Get().Transaction(func(tx *gorm.DB) error {
		userToken := &dao.UserToken{}
		err = tx.Where("wallet_address = ?", address).Preload("LockedTokens").First(userToken).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			// save a record if locked token is zero
			// mostly used in test
			if tokenAmount == 0 {
				tx.Save(userToken)
			}
		}
		lockedAmount := int32(0)
		lockedNum := 0
		for _, locked := range userToken.LockedTokens {
			if locked.TemporaryAddress == tempAddress {
				return errors.New("user token is locked")
			}
			if time.Since(locked.CreatedAt) < maxLockTime {
				lockedNum++
				lockedAmount += locked.TokenAmount
			} else {
				err = tx.Delete(locked).Error
				if err != nil {
					return err
				}
			}
		}
		if lockedNum >= maxPlayerPerAddress {
			return errors.New("cannot lock token, locked temporary address num exceeds limit")
		}
		if userToken.TokenAmount < tokenAmount+lockedAmount {
			return errors.New("user token amount is not enough")
		}
		newLocked := &dao.LockedUserToken{
			UserTokenID:      userToken.ID,
			TokenAmount:      tokenAmount,
			TemporaryAddress: tempAddress,
		}
		err = tx.Save(newLocked).Error
		if err != nil {
			return err
		}
		return nil
	})
}

func UnlockUserToken(ctx context.Context, address string, tempAddress string) (err error) {
	return Get().Transaction(func(tx *gorm.DB) error {
		userToken := &dao.UserToken{}
		err = tx.Where("wallet_address = ?", address).Preload("LockedTokens").First(userToken).Error
		if err != nil {
			return err
		}
		for _, locked := range userToken.LockedTokens {
			if locked.TemporaryAddress == tempAddress {
				err = tx.Delete(locked).Error
				if err != nil {
					return err
				}
			}
			if time.Since(locked.CreatedAt) > maxLockTime {
				err = tx.Delete(locked).Error
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func BattleResultSettlement(game *dao.Game) error {
	// game aborted when init
	if game.Status == proto.GameStatus_GAME_INIT {
		for _, pr := range game.Players {
			log.Debugw("unlock player token", "wallet addr", pr.WalletAddress, "temp addr", pr.TemporaryAddress)
			err := UnlockUserToken(context.Background(), pr.WalletAddress, pr.TemporaryAddress)
			if err != nil {
				log.Errorw("cannot unlock user token", "err", err, "wallet addr", pr.WalletAddress, "temp addr", pr.TemporaryAddress)
			}
		}
		return nil
	}
	if game.GameResult == nil {
		return errors.New("game result is nil")
	}
	reward := game.GameResult.BattleReward
	if reward == nil {
		return errors.New("game result battle reward is nil")
	}
	return Get().Transaction(func(tx *gorm.DB) error {
		for _, pr := range reward.PlayerRewards {
			userToken := &dao.UserToken{}
			err := tx.Where("wallet_address = ?", pr.WalletAddress).Preload("LockedTokens").First(userToken).Error
			if err != nil {
				return fmt.Errorf("find user token record from db failed, game id: %d, address: %s, err: %w", game.ID, pr.WalletAddress, err)
			}
			userToken.TokenAmount += pr.TokenChange
			userToken.Points += pr.PointChange
			for _, locked := range userToken.LockedTokens {
				if locked.TemporaryAddress == pr.TemporaryAddress {
					err = tx.Delete(locked).Error
					if err != nil {
						return err
					}
				}
			}
			err = tx.Save(userToken).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func GetPlayerToken(ctx context.Context, address string) (*dao.UserToken, error) {
	var userToken dao.UserToken
	err := Get().Where("wallet_address = ?", address).Preload("LockedTokens").First(&userToken).Error
	if err != nil {
		return nil, err
	}
	// filter locked tokens by time
	lockedTokens := make([]*dao.LockedUserToken, 0)
	for _, locked := range userToken.LockedTokens {
		if time.Since(locked.CreatedAt) < maxLockTime {
			lockedTokens = append(lockedTokens, locked)
		}
	}
	userToken.LockedTokens = lockedTokens
	return &userToken, nil
}

func SetLockedTokenGameID(ctx context.Context, walletAddress, temporaryAddress string, gameID uint) error {
	return Get().Transaction(func(tx *gorm.DB) error {
		userToken := &dao.UserToken{}
		err := tx.Where("wallet_address = ?", walletAddress).Preload("LockedTokens").First(userToken).Error
		if err != nil {
			return err
		}
		for _, locked := range userToken.LockedTokens {
			if locked.TemporaryAddress == temporaryAddress {
				locked.GameID = gameID
				return tx.Save(locked).Error
			}
		}
		return errors.New("locked token not found")
	})
}
