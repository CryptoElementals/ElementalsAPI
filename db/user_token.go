package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CryptoElementals/common/battlereward"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

const maxLockTimeForQueue = 10 * time.Minute
const maxLockTimeForTournament = 30 * time.Minute
const maxPlayerPerAddress = 3

func SaveUserToken(tokens ...dao.UserToken) error {
	return Get().Save(&tokens).Error
}

// EnsureUserTokenByPlayerID creates an empty user_token row when missing.
func EnsureUserTokenByPlayerID(playerID int64) (*dao.UserToken, error) {
	return EnsureUserTokenByPlayerIDTx(Get(), playerID)
}

// EnsureUserTokenByPlayerIDTx creates an empty user_token row when missing in an existing DB session.
func EnsureUserTokenByPlayerIDTx(tx *gorm.DB, playerID int64) (*dao.UserToken, error) {
	var userToken dao.UserToken
	err := tx.Where("player_id = ?", playerID).First(&userToken).Error
	if err == nil {
		return &userToken, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	userToken = dao.UserToken{
		PlayerId:    playerID,
		Points:      0,
		TokenAmount: 0,
	}
	if err := tx.Create(&userToken).Error; err != nil {
		return nil, err
	}
	return &userToken, nil
}

// CreditUserTokenAmount adds delta to token_amount for the player (creates row if missing).
func CreditUserTokenAmount(playerID int64, delta int32) (*dao.UserToken, error) {
	var updated *dao.UserToken
	err := Get().Transaction(func(tx *gorm.DB) error {
		token, err := EnsureUserTokenByPlayerIDTx(tx, playerID)
		if err != nil {
			return err
		}
		res := tx.Model(&dao.UserToken{}).
			Where("id = ?", token.ID).
			Update("token_amount", gorm.Expr("token_amount + ?", delta))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("user token not found")
		}
		updated, err = EnsureUserTokenByPlayerIDTx(tx, playerID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// SetUserTokenAmount sets token_amount for the player (creates row if missing).
func SetUserTokenAmount(playerID int64, tokenAmount int32) (*dao.UserToken, error) {
	var updated *dao.UserToken
	err := Get().Transaction(func(tx *gorm.DB) error {
		token, err := EnsureUserTokenByPlayerIDTx(tx, playerID)
		if err != nil {
			return err
		}
		res := tx.Model(&dao.UserToken{}).
			Where("id = ?", token.ID).
			Update("token_amount", tokenAmount)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("user token not found")
		}
		updated, err = EnsureUserTokenByPlayerIDTx(tx, playerID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// DeductUserTokenForTournamentEntry deducts tokenAmount directly from user_tokens for tournament registration.
// Returns "user token amount is not enough" when balance is insufficient (or row missing).
func DeductUserTokenForTournamentEntry(ctx context.Context, playerID int64, tokenAmount int32) error {
	if tokenAmount <= 0 {
		return nil
	}
	return Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return DeductUserTokenForTournamentEntryTx(tx, playerID, tokenAmount)
	})
}

// DeductUserTokenForTournamentEntryTx deducts tokenAmount directly from user_tokens in an existing tx.
func DeductUserTokenForTournamentEntryTx(tx *gorm.DB, playerID int64, tokenAmount int32) error {
	if tokenAmount <= 0 {
		return nil
	}
	res := tx.Model(&dao.UserToken{}).
		Where("player_id = ? AND token_amount >= ?", playerID, tokenAmount).
		Update("token_amount", gorm.Expr("token_amount - ?", tokenAmount))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("user token amount is not enough")
	}
	return nil
}

// RefundUserTokenForTournamentEntryTx refunds tokenAmount back to user_tokens in an existing tx.
func RefundUserTokenForTournamentEntryTx(tx *gorm.DB, playerID int64, tokenAmount int32) error {
	if tokenAmount <= 0 {
		return nil
	}
	res := tx.Model(&dao.UserToken{}).
		Where("player_id = ?", playerID).
		Update("token_amount", gorm.Expr("token_amount + ?", tokenAmount))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("user token not found")
	}
	return nil
}

func RecordTournamentEntryLedgerTx(
	tx *gorm.DB,
	tournamentID string,
	playerID int64,
	tempAddress string,
	amount int32,
	direction dao.TournamentEntryLedgerDirection,
	reason string,
) error {
	row := &dao.TournamentEntryLedger{
		TournamentID: tournamentID,
		PlayerID:     playerID,
		TempAddress:  strings.ToLower(strings.TrimSpace(tempAddress)),
		Amount:       amount,
		Direction:    direction,
		Reason:       reason,
	}
	return tx.Create(row).Error
}

func LockUserToken(ctx context.Context, playerId int64, tempAddress string, tokenAmount int32, tournamentID string) (err error) {
	return Get().Transaction(func(tx *gorm.DB) error {
		// resolve user by address
		userToken := &dao.UserToken{}
		err = tx.Where("player_id = ?", playerId).Preload("LockedTokens").First(userToken).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return errors.New("user token amount is not enough")
			}
		}
		lockedAmount := int32(0)
		lockedNum := 0
		for _, locked := range userToken.LockedTokens {
			maxLockTime := maxLockTimeForQueue
			if locked.TournamentID != "" {
				maxLockTime = maxLockTimeForTournament
			}

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
				if locked.TournamentID == "" {
					if locked.GameID == 0 {
						return errors.New("user token is locked")
					}
					log.Infow("LockUserToken called but game id != 0", "game id", locked.GameID)
					// already locked by some game
					// check if game is still running
					gameInfo := &dao.Game{}
					err := tx.Where("id = ? and status != ?", locked.GameID, proto.GameStatus_GAME_END).Find(gameInfo).Error
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

				if locked.TournamentID != "" {
					// Check whether tournament has ended; if ended/stale, cleanup lock.
					tournament := &dao.Tournament{}
					err := tx.Where(
						"tournament_id = ? AND status IN ?",
						locked.TournamentID,
						[]dao.TournamentStatus{
							dao.TournamentStatusRegistrationOpen,
							dao.TournamentStatusInProgress,
						},
					).First(tournament).Error
					if err == gorm.ErrRecordNotFound {
						if derr := tx.Delete(locked).Error; derr != nil {
							return derr
						}
						lockedNum--
						lockedAmount -= locked.TokenAmount
						continue
					}
					if err != nil {
						return err
					}
					return errors.New("user token is locked")
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
			TournamentID:     tournamentID,
		}
		err = tx.Save(newLocked).Error
		if err != nil {
			return err
		}
		return nil
	})
}

func LockUserTokenForContinue(ctx context.Context, playerIds []int64, tempAddresses []string, tokenAmount int32, gameID int64) (err error) {
	return Get().Transaction(func(tx *gorm.DB) error {
		for i := range playerIds {
			playerId := playerIds[i]
			tempAddress := tempAddresses[i]
			userToken := &dao.UserToken{}
			err = tx.Where("player_id = ?", playerId).Preload("LockedTokens").First(userToken).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			}
			lockedAmount := int32(0)
			lockedNum := 0
			for _, locked := range userToken.LockedTokens {
				if time.Since(locked.CreatedAt) < maxLockTimeForQueue {
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

func UnlockUserToken(ctx context.Context, playerId int64, tempAddress string, isTournament bool) (err error) {
	return Get().Transaction(func(tx *gorm.DB) error {
		userToken := &dao.UserToken{}
		err = tx.Where("player_id = ?", playerId).Preload("LockedTokens").First(userToken).Error
		if err != nil {
			return err
		}

		maxLockTime := maxLockTimeForQueue
		if isTournament {
			maxLockTime = maxLockTimeForTournament
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

func UnlockUserTokenByGameID(ctx context.Context, gameID int64) error {
	return Get().Transaction(func(tx *gorm.DB) error {
		cnt, err := gorm.G[dao.LockedUserToken](tx).Where("game_id = ?", gameID).Delete(ctx)
		if err != nil {
			return err
		}
		log.Debugw("UnlockUserTokenByGameID", "unlocked cnt", cnt)
		return nil
	})
}

// BattleResultSettlement applies PVP economy updates for gr.GameID. gr must be non-nil (caller loads it).
// Inside the transaction: lock games row, load game_args only (via games.game_args_id), not turns/players.
// When skippedDuplicate is true, economy was already applied (battle_rewards row existed); callers may skip side effects that are not safe to repeat.
func BattleResultSettlement(gr *dao.GameResult) (skippedDuplicate bool, err error) {
	if gr == nil {
		return false, errors.New("battle result settlement: game result is nil")
	}
	gameID := gr.GameID
	if gameID == 0 {
		return false, nil
	}
	if gr.GameResultType == proto.GameResultType_GAME_ABORTED {
		log.Debugw("unlock player token caused by abort outcome", "game id", gameID)
		return false, UnlockUserTokenByGameID(context.Background(), gameID)
	}
	var skipped bool
	if err := Get().Transaction(func(tx *gorm.DB) error {
		if err := LockGameForUpdateTx(tx, gameID); err != nil {
			return err
		}
		ga, err := LoadGameArgsByGameIDTx(tx, gameID)
		if err != nil {
			return err
		}
		baseStake := ga.BaseStake
		if baseStake <= 0 {
			return fmt.Errorf("game_args.base_stake must be positive (game id %d)", gameID)
		}
		exists, err := BattleRewardPVPExistsForGame(tx, gameID)
		if err != nil {
			return err
		}
		if exists {
			log.Debugw("battle settlement skipped: battle_rewards row already exists for game", "game_id", gameID)
			skipped = true
			return nil
		}
		reward, err := EnsureBattleRewardPVPLoadedOrCreated(tx, gameID, gr)
		if err != nil {
			return err
		}
		battlereward.ComputeBattleRewardAmounts(gr, reward, ga)

		type playerReward struct {
			playerId    int64
			tempAddr    string
			tokenChange int32
			pointChange int32
		}
		var toProcess []playerReward
		infos := gr.PlayerResultInfos
		for _, pr := range reward.PlayerRewards {
			if pr == nil {
				continue
			}
			tempAddr := TemporaryAddressForPlayer(infos, pr.PlayerId)
			toProcess = append(toProcess, playerReward{
				playerId:    pr.PlayerId,
				tempAddr:    tempAddr,
				tokenChange: pr.TokenChange,
				pointChange: pr.PointChange,
			})
		}

		for _, pr := range reward.PlayerRewards {
			if pr == nil {
				continue
			}
			if err := tx.Model(&dao.PlayerReward{}).Where("id = ?", pr.ID).
				Updates(map[string]any{
					"token_change": pr.TokenChange,
					"point_change": pr.PointChange,
				}).Error; err != nil {
				return fmt.Errorf("update player_reward id %d: %w", pr.ID, err)
			}
		}
		if err := tx.Model(&dao.BattleRewardPVP{}).Where("id = ?", reward.ID).
			Update("system_fee", reward.SystemFee).Error; err != nil {
			return fmt.Errorf("update battle_reward id %d: %w", reward.ID, err)
		}
		for _, pr := range toProcess {
			if err := tx.Where("temporary_address = ?", pr.tempAddr).
				Delete(&dao.LockedUserToken{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&dao.UserToken{}).Where("player_id = ?", pr.playerId).
				Updates(map[string]any{
					"token_amount": gorm.Expr("token_amount + ?", pr.tokenChange),
					"points":       gorm.Expr("points + ?", pr.pointChange),
				}).Error; err != nil {
				return fmt.Errorf("update user token failed, game id: %d, player id: %d, err: %w", gameID, pr.playerId, err)
			}
		}
		return nil
	}); err != nil {
		return false, fmt.Errorf("battle result settlement: game id: %d: %w", gameID, err)
	}

	return skipped, nil
}

func GetPlayerToken(ctx context.Context, playerId int64) (*dao.UserToken, error) {
	var userToken dao.UserToken
	err := Get().Where("player_id = ?", playerId).Preload("LockedTokens").First(&userToken).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dao.UserToken{
				PlayerId:     playerId,
				Points:       0,
				TokenAmount:  0,
				LockedTokens: []*dao.LockedUserToken{},
			}, nil
		}
		return nil, err
	}
	// filter locked tokens by time
	lockedTokens := make([]*dao.LockedUserToken, 0)
	for _, locked := range userToken.LockedTokens {
		if time.Since(locked.CreatedAt) < maxLockTimeForQueue {
			lockedTokens = append(lockedTokens, locked)
		}
	}
	userToken.LockedTokens = lockedTokens
	return &userToken, nil
}

// DeleteAllLockedUserTokens removes all rows from locked_user_tokens (operator/tools; matchmaking reset).
func DeleteAllLockedUserTokens() (rowsAffected int64, err error) {
	res := Get().Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().
		Where("1 = 1").Delete(&dao.LockedUserToken{})
	return res.RowsAffected, res.Error
}

// GetQueueLockedGameID returns the active PVP queue game id from locked_user_tokens for the given address, if any.
func GetQueueLockedGameID(ctx context.Context, playerId int64, temporaryAddress string) (int64, error) {
	ut, err := GetPlayerToken(ctx, playerId)
	if err != nil {
		return 0, err
	}
	temp := strings.ToLower(strings.TrimSpace(temporaryAddress))
	for _, locked := range ut.LockedTokens {
		if locked == nil {
			continue
		}
		if locked.TournamentID != "" {
			continue
		}
		if strings.ToLower(locked.TemporaryAddress) != temp {
			continue
		}
		if locked.GameID == 0 {
			continue
		}
		return locked.GameID, nil
	}
	return 0, nil
}

func SetLockedTokenGameID(ctx context.Context, playerId int64, temporaryAddress string, gameID int64) error {
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
