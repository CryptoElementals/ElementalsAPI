package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

const maxLockTime = 10 * time.Minute
const maxPlayerPerAddress = 3

func SaveUserToken(tokens ...dao.UserToken) error {
	return Get().Save(&tokens).Error
}

func LockUserToken(ctx context.Context, playerId int64, tempAddress string, tokenAmount int32) (err error) {
	return Get().Transaction(func(tx *gorm.DB) error {
		// resolve user by address
		profile, perr := GetUserProfileByPlayerID(strconv.FormatInt(playerId, 10))
		if perr != nil {
			return perr
		}
		userToken := &dao.UserToken{}
		err = tx.Where("player_id = ?", playerId).Preload("LockedTokens").First(userToken).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			// save a record if locked token is zero
			// mostly used in test
			if tokenAmount == 0 {
				userToken.PlayerId = profile.PlayerID
				tx.Save(userToken)
			}
		}
		lockedAmount := int32(0)
		lockedNum := 0
		for _, locked := range userToken.LockedTokens {
			if time.Since(locked.CreatedAt) < maxLockTime {
				lockedNum++
				lockedAmount += locked.TokenAmount
			} else {
				err = tx.Delete(locked).Error
				if err != nil {
					return err
				}
				continue
			}
			if locked.TemporaryAddress == tempAddress {
				if locked.GameID == 0 {
					return errors.New("user token is locked")
				}
				log.Infow("LockUserToken called but game id != 0", "game id", locked.GameID)
				// already locked by some game
				// check if game is still running
				gameInfo := &dao.Game{}
				err := tx.Where("id = ? and status != ? and status != ?", locked.GameID, proto.GameStatus_GAME_END, proto.GameStatus_GAME_ABORTED).Find(gameInfo).Error
				if err == nil {
					// game still running
					return errors.New("user token is locked")
				}
				if err != gorm.ErrRecordNotFound {
					return err
				}
				// game stopped, delete the record
				log.Infow("locked game info not found", "game id", locked.GameID)
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

func LockUserTokenForContinue(ctx context.Context, playerIds []int64, tempAddresses []string, tokenAmount int32, gameID uint) (err error) {
	return Get().Transaction(func(tx *gorm.DB) error {
		for i := range playerIds {
			playerId := playerIds[i]
			tempAddress := tempAddresses[i]
			profile, perr := GetUserProfileByPlayerID(strconv.FormatInt(playerId, 10))
			if perr != nil {
				return perr
			}
			userToken := &dao.UserToken{}
			err = tx.Where("player_id = ?", playerId).Preload("LockedTokens").First(userToken).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
				// save a record if locked token is zero
				// mostly used in test
				if tokenAmount == 0 {
					userToken.PlayerId = profile.PlayerID
					tx.Save(userToken)
				}
			}
			lockedAmount := int32(0)
			lockedNum := 0
			for _, locked := range userToken.LockedTokens {
				if time.Since(locked.CreatedAt) < maxLockTime {
					lockedNum++
					lockedAmount += locked.TokenAmount
				} else {
					err = tx.Delete(locked).Error
					if err != nil {
						return err
					}
					continue
				}
				if locked.TemporaryAddress == tempAddress {
					return errors.New("user token is locked")
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
				GameID:           gameID,
			}
			err = tx.Save(newLocked).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func UnlockUserToken(ctx context.Context, playerId int64, tempAddress string) (err error) {
	return Get().Transaction(func(tx *gorm.DB) error {
		_, perr := GetUserProfileByPlayerID(strconv.FormatInt(playerId, 10))
		if perr != nil {
			return perr
		}
		userToken := &dao.UserToken{}
		err = tx.Where("player_id = ?", playerId).Preload("LockedTokens").First(userToken).Error
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

func UnlockUserTokenByGameID(ctx context.Context, gameID uint) error {
	return Get().Transaction(func(tx *gorm.DB) error {
		cnt, err := gorm.G[dao.LockedUserToken](tx).Where("game_id = ?", gameID).Delete(ctx)
		if err != nil {
			return err
		}
		log.Debugw("UnlockUserTokenByGameID", "unlocked cnt", cnt)
		return nil
	})
}

func BattleResultSettlement(game *dao.Game, bots map[types.PlayerAddress]struct{}) error {
	// game aborted when init
	if game.Status == proto.GameStatus_GAME_ABORTED {
		log.Debugw("unlock player token caused by abort", "game id", game.ID)
		return UnlockUserTokenByGameID(context.Background(), game.ID)
	}
	if game.GameResult == nil {
		return errors.New("game result is nil")
	}
	reward := game.GameResult.BattleReward
	if reward == nil {
		return errors.New("game result battle reward is nil")
	}
	// Build list of non-bot player rewards to process
	type playerReward struct {
		playerId    int64
		tempAddr    string
		tokenChange int32
		pointChange int32
	}
	var toProcess []playerReward
	for _, pr := range reward.PlayerRewards {
		if _, ok := bots[types.PlayerAddress{
			Id:               pr.PlayerId,
			TemporaryAddress: pr.TemporaryAddress,
		}]; ok {
			continue
		}
		toProcess = append(toProcess, playerReward{
			playerId:    pr.PlayerId,
			tempAddr:    pr.TemporaryAddress,
			tokenChange: pr.TokenChange,
			pointChange: pr.PointChange,
		})
	}

	// First transaction: delete locked tokens for each player
	if err := Get().Transaction(func(tx *gorm.DB) error {
		for _, pr := range toProcess {
			if err := tx.Where("temporary_address = ?", pr.tempAddr).
				Delete(&dao.LockedUserToken{}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("battle result settlement: delete locked tokens failed, game id: %d: %w", game.ID, err)
	}

	// Second transaction: update token_amount and points using += / -= in SQL
	if err := Get().Transaction(func(tx *gorm.DB) error {
		for _, pr := range toProcess {
			err := tx.Model(&dao.UserToken{}).Where("player_id = ?", pr.playerId).
				Updates(map[string]any{
					"token_amount": gorm.Expr("token_amount + ?", pr.tokenChange),
					"points":       gorm.Expr("points + ?", pr.pointChange),
				}).Error
			if err != nil {
				return fmt.Errorf("update user token failed, game id: %d, player id: %d, err: %w", game.ID, pr.playerId, err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("battle result settlement: update token amount and points failed, game id: %d: %w", game.ID, err)
	}

	return nil
}

// CollectDailyRewardByPlayerID 在单个事务中为用户发放每日奖励 Token 并更新领取时间
func CollectDailyRewardByPlayerID(playerID string, dailyRewardTokens int32) error {
	id, err := strconv.ParseInt(playerID, 10, 64)
	if err != nil {
		return err
	}

	now := time.Now()

	return Get().Transaction(func(tx *gorm.DB) error {
		// 查询或创建用户的 Token 记录
		var userToken dao.UserToken
		if err := tx.Where("player_id = ?", id).First(&userToken).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				userToken = dao.UserToken{
					PlayerId:    id,
					Points:      0,
					TokenAmount: dailyRewardTokens,
				}
				if err := tx.Create(&userToken).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			userToken.TokenAmount += dailyRewardTokens
			if err := tx.Save(&userToken).Error; err != nil {
				return err
			}
		}

		// 同一事务内更新用户每日奖励领取时间
		if err := tx.Model(&dao.UserProfile{}).
			Where("player_id = ?", id).
			Update("collected_reward_at", now).Error; err != nil {
			return err
		}

		return nil
	})
}

func GetPlayerToken(ctx context.Context, playerId int64) (*dao.UserToken, error) {
	var userToken dao.UserToken
	err := Get().Where("player_id = ?", playerId).Preload("LockedTokens").First(&userToken).Error
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

func SetLockedTokenGameID(ctx context.Context, playerId int64, temporaryAddress string, gameID uint) error {
	return Get().Transaction(func(tx *gorm.DB) error {
		userToken := &dao.UserToken{}
		err := tx.Where("player_id = ?", playerId).Preload("LockedTokens").First(userToken).Error
		if err != nil {
			return err
		}
		for _, locked := range userToken.LockedTokens {
			if locked.TemporaryAddress == temporaryAddress {
				locked.GameID = gameID
				return tx.Save(locked).Error
			}
		}
		return ErrNotFound
	})
}
