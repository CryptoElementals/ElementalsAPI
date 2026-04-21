package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// TournamentApplyMatchRewardsTx upserts one player's latest win reward snapshot in tournament_rewards
// (token_change/point_change from tournament_tier_reward_configs for pool size + round tier).
// Wallet is not updated here; settlement happens when eliminated/champion.
func TournamentApplyMatchRewardsTx(tx *gorm.DB, tournamentID string, roundNo, matchNo uint32,
	playerID int64, playerTemp string,
) error {
	playerTemp = strings.ToLower(strings.TrimSpace(playerTemp))

	if playerID == 0 || playerTemp == "" {
		return fmt.Errorf("tournament reward: invalid player for %s r%d m%d", tournamentID, roundNo, matchNo)
	}

	totalParticipants, err := TournamentCountParticipantsForPool(tournamentID)
	if err != nil {
		return fmt.Errorf("tournament reward: count participants: %w", err)
	}
	rewardToken, rewardPoint, err := TournamentRoundReward(tx, int32(totalParticipants), roundNo)
	if err != nil {
		return fmt.Errorf("tournament reward: round reward config: %w", err)
	}

	rewardRow := dao.TournamentReward{
		TournamentID: tournamentID,
		RoundNo:      roundNo,
		MatchNo:      matchNo,
		PlayerID:     playerID,
		TempAddress:  playerTemp,
		TokenChange:  rewardToken,
		PointChange:  rewardPoint,
	}
	deltaToken := rewardToken
	deltaPoint := rewardPoint

	var previous dao.TournamentReward
	prevErr := tx.Where("tournament_id = ? AND player_id = ? AND temp_address = ?",
		tournamentID, playerID, playerTemp).
		Order("round_no DESC, match_no DESC, id DESC").
		First(&previous).Error
	switch {
	case prevErr == nil:
		deltaToken = rewardToken - previous.TokenChange
		deltaPoint = rewardPoint - previous.PointChange
		previous.RoundNo = roundNo
		previous.MatchNo = matchNo
		previous.TokenChange = rewardToken
		previous.PointChange = rewardPoint
		if err := tx.Save(&previous).Error; err != nil {
			return fmt.Errorf("tournament reward update latest snapshot: %w", err)
		}
	case errors.Is(prevErr, gorm.ErrRecordNotFound):
		if err := tx.Create(&rewardRow).Error; err != nil {
			return fmt.Errorf("tournament reward insert latest snapshot: %w", err)
		}
	default:
		return fmt.Errorf("tournament reward: load previous winner reward: %w", prevErr)
	}

	log.Infow("tournament: player win reward snapshot updated",
		"tournament_id", tournamentID, "round_no", roundNo, "match_no", matchNo,
		"player_id", playerID, "player_temp", playerTemp,
		"total_participants", totalParticipants, "reward_token", rewardToken, "reward_point", rewardPoint,
		"delta_token_vs_previous_snapshot", deltaToken, "delta_point_vs_previous_snapshot", deltaPoint)

	return nil
}

// TournamentSettlePlayerRewardToWalletTx pays the player's latest snapshot reward to wallet.
// If player has no win snapshot, it is a no-op.
func TournamentSettlePlayerRewardToWalletTx(tx *gorm.DB, tournamentID string, playerID int64, tempAddress string) error {
	tempAddress = strings.ToLower(strings.TrimSpace(tempAddress))
	if tournamentID == "" || playerID == 0 || tempAddress == "" {
		return fmt.Errorf("invalid settlement args")
	}
	var row dao.TournamentReward
	err := tx.Where("tournament_id = ? AND player_id = ? AND temp_address = ?", tournamentID, playerID, tempAddress).
		Order("round_no DESC, match_no DESC, id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.TokenChange == 0 && row.PointChange == 0 {
		return nil
	}
	res := tx.Model(&dao.UserToken{}).Where("player_id = ?", playerID).
		Updates(map[string]any{
			"token_amount": gorm.Expr("token_amount + ?", row.TokenChange),
			"points":       gorm.Expr("points + ?", row.PointChange),
		})
	if res.Error != nil {
		return fmt.Errorf("tournament settle wallet player %d: %w", playerID, res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("tournament settle: user_token missing for player_id %d", playerID)
	}
	log.Infow("tournament: player reward settled to wallet",
		"tournament_id", tournamentID, "player_id", playerID, "temp_address", tempAddress,
		"token_change", row.TokenChange, "point_change", row.PointChange,
		"source_round_no", row.RoundNo, "source_match_no", row.MatchNo)
	return nil
}

// TournamentSumPlayerRewardTotalsTx returns summed token_change and point_change for one player in a tournament (all matches).
func TournamentSumPlayerRewardTotalsTx(tx *gorm.DB, tournamentID string, playerID int64, tempAddress string) (tokenSum int32, pointSum int32, err error) {
	tempAddress = strings.ToLower(strings.TrimSpace(tempAddress))
	var agg struct {
		Token int64 `gorm:"column:token"`
		Point int64 `gorm:"column:point"`
	}
	err = tx.Model(&dao.TournamentReward{}).
		Select("COALESCE(SUM(token_change), 0) AS token, COALESCE(SUM(point_change), 0) AS point").
		Where("tournament_id = ? AND player_id = ? AND temp_address = ?", tournamentID, playerID, tempAddress).
		Scan(&agg).Error
	if err != nil {
		return 0, 0, err
	}
	return int32(agg.Token), int32(agg.Point), nil
}

// TournamentCumulativeBattleRewardProtoTx builds a BattleReward with winner then loser cumulative tournament totals (after current tx writes).
func TournamentCumulativeBattleRewardProtoTx(tx *gorm.DB, tournamentID string, winPID int64, winTemp string, losePID int64, loseTemp string) (*proto.BattleReward, error) {
	winTemp = strings.ToLower(strings.TrimSpace(winTemp))
	loseTemp = strings.ToLower(strings.TrimSpace(loseTemp))
	wt, wp, err := TournamentSumPlayerRewardTotalsTx(tx, tournamentID, winPID, winTemp)
	if err != nil {
		return nil, err
	}
	lt, lp, err := TournamentSumPlayerRewardTotalsTx(tx, tournamentID, losePID, loseTemp)
	if err != nil {
		return nil, err
	}

	return &proto.BattleReward{
		SystemFee: 0,
		PlayerRewards: []*proto.PlayerReward{
			{PlayerId: winPID, TemporaryAddress: winTemp, TokenChange: wt, PointChange: wp},
			{PlayerId: losePID, TemporaryAddress: loseTemp, TokenChange: lt, PointChange: lp},
		},
	}, nil
}

func TournamentRoundReward(tx *gorm.DB, totalParticipants int32, roundNo uint32) (token int32, point int32, err error) {
	if roundNo == 0 {
		return 0, 0, nil
	}

	totalPlayerCount := int32(64)
	switch {
	case totalParticipants < 128:
		totalPlayerCount = 64
	case totalParticipants < 256:
		totalPlayerCount = 128
	case totalParticipants < 512:
		totalPlayerCount = 256
	case totalParticipants < 1024:
		totalPlayerCount = 512
	case totalParticipants < 2048:
		totalPlayerCount = 1024
	case totalParticipants < 4096:
		totalPlayerCount = 2048
	case totalParticipants < 8192:
		totalPlayerCount = 4096
	default:
		totalPlayerCount = 8192
	}

	var tierCfg dao.TournamentTierRewardConfig
	if err := tx.Where("total_player_count = ? AND tier_no = ?", totalPlayerCount, int32(roundNo)).
		First(&tierCfg).Error; err != nil {
		return 0, 0, err
	}
	return tierCfg.RewardToken, tierCfg.Point, nil
}
