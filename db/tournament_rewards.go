package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TournamentApplyMatchRewardsTx upserts the winner's tournament_rewards row for this bracket match
// (token_change/point_change from tournament_tier_reward_configs for pool size + round tier),
// then applies the wallet delta equal to (new reward - previous reward) for idempotent retries.
func TournamentApplyMatchRewardsTx(tx *gorm.DB, tournamentID string, roundNo, matchNo uint32,
	winnerID int64, winnerTemp string,
) error {
	winnerTemp = strings.ToLower(strings.TrimSpace(winnerTemp))

	if winnerID == 0 || winnerTemp == "" {
		return fmt.Errorf("tournament reward: invalid winner for %s r%d m%d", tournamentID, roundNo, matchNo)
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
		PlayerID:     winnerID,
		TempAddress:  winnerTemp,
		TokenChange:  rewardToken,
		PointChange:  rewardPoint,
	}
	deltaToken := rewardToken
	deltaPoint := rewardPoint

	var previous dao.TournamentReward
	prevErr := tx.Where("tournament_id = ? AND round_no = ? AND match_no = ? AND player_id = ? AND temp_address = ?",
		tournamentID, roundNo, matchNo, winnerID, winnerTemp).First(&previous).Error
	switch {
	case prevErr == nil:
		deltaToken = rewardToken - previous.TokenChange
		deltaPoint = rewardPoint - previous.PointChange
	case errors.Is(prevErr, gorm.ErrRecordNotFound):
		// first time reward write for this winner+match
	default:
		return fmt.Errorf("tournament reward: load previous winner reward: %w", prevErr)
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tournament_id"},
			{Name: "round_no"},
			{Name: "match_no"},
			{Name: "player_id"},
			{Name: "temp_address"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"token_change", "point_change", "updated_at"}),
	}).Create(&rewardRow).Error; err != nil {
		return fmt.Errorf("tournament reward upsert: %w", err)
	}

	log.Infow("tournament: winner reward resolved from tier config",
		"tournament_id", tournamentID, "round_no", roundNo, "match_no", matchNo,
		"winner_id", winnerID, "winner_temp", winnerTemp,
		"total_participants", totalParticipants, "reward_token", rewardToken, "reward_point", rewardPoint,
		"delta_token", deltaToken, "delta_point", deltaPoint)

	if deltaToken != 0 || deltaPoint != 0 {
		res := tx.Model(&dao.UserToken{}).Where("player_id = ?", winnerID).
			Updates(map[string]any{
				"token_amount": gorm.Expr("token_amount + ?", deltaToken),
				"points":       gorm.Expr("points + ?", deltaPoint),
			})
		if res.Error != nil {
			return fmt.Errorf("tournament reward wallet player %d: %w", winnerID, res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("tournament reward: user_token missing for player_id %d", winnerID)
		}
	}

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
func TournamentCumulativeBattleRewardProtoTx(tx *gorm.DB, tournamentID string, winPID int64, winTemp string, losePID int64, loseTemp string, roundNo uint32) (*proto.BattleReward, error) {
	winTemp = strings.ToLower(strings.TrimSpace(winTemp))
	loseTemp = strings.ToLower(strings.TrimSpace(loseTemp))
	totalParticipants, err := TournamentCountParticipantsForPool(tournamentID)
	if err != nil {
		return nil, err
	}

	lt, lp := int32(0), int32(0)
	wt, wp := int32(0), int32(0)
	if roundNo > 0 {
		lastRoundNo := roundNo - 1
		lt, lp, err = TournamentRoundReward(tx, int32(totalParticipants), lastRoundNo)
		if err != nil {
			return nil, err
		}
	}
	wt, wp, err = TournamentRoundReward(tx, int32(totalParticipants), roundNo)
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
