package db

import (
	"errors"
	"fmt"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

// EnsureBattleRewardPVPLoadedOrCreated loads battle_rewards + player_rewards for gameID, or inserts
// one BattleRewardPVP and one PlayerReward per PlayerResultInfo (amounts zero). Used only at lobby settlement.
func EnsureBattleRewardPVPLoadedOrCreated(tx *gorm.DB, gameID int64, gr *dao.GameResult) (*dao.BattleRewardPVP, error) {
	if gr == nil || len(gr.PlayerResultInfos) == 0 {
		return nil, fmt.Errorf("game result or player result infos missing (game id %d)", gameID)
	}
	var br dao.BattleRewardPVP
	err := tx.Where("game_id = ?", gameID).Preload("PlayerRewards").First(&br).Error
	if err == nil {
		return &br, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	br = dao.BattleRewardPVP{GameID: gameID}
	if err := tx.Session(&gorm.Session{FullSaveAssociations: false}).Omit("PlayerRewards").Create(&br).Error; err != nil {
		return nil, err
	}
	for _, pri := range gr.PlayerResultInfos {
		if pri == nil {
			continue
		}
		pr := &dao.PlayerReward{
			BattleRewardID: br.ID,
			PlayerId:       pri.PlayerId,
		}
		if err := tx.Create(pr).Error; err != nil {
			return nil, err
		}
	}
	var out dao.BattleRewardPVP
	if err := tx.Where("id = ?", br.ID).Preload("PlayerRewards").First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// BattleRewardPVPExistsForGame reports whether a battle_rewards row exists for this game_id
// (settlement created the skeleton). Used to avoid double settlement.
func BattleRewardPVPExistsForGame(tx *gorm.DB, gameID int64) (bool, error) {
	var n int64
	if err := tx.Model(&dao.BattleRewardPVP{}).Where("game_id = ?", gameID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// LoadBattleRewardPVPByGameID loads battle_rewards for a game (e.g. after settlement for pubsub).
func LoadBattleRewardPVPByGameID(gameID int64) (*dao.BattleRewardPVP, error) {
	var br dao.BattleRewardPVP
	if err := Get().Where("game_id = ?", gameID).Preload("PlayerRewards").First(&br).Error; err != nil {
		return nil, err
	}
	return &br, nil
}
