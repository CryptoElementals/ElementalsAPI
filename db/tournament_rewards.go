package db

import (
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// Tournament bracket match rewards (not PVP battle_rewards / player_rewards).
// todo: hardcode rewards for now, later updated
const (
	tournamentRewardWinToken  int32 = 1000
	tournamentRewardWinPoint  int32 = 20
	tournamentRewardTieToken  int32 = 0
	tournamentRewardTiePoint  int32 = 10
	tournamentRewardLossToken int32 = -1000
	tournamentRewardLossPoint int32 = 5
)

func protoBattleRewardFromTournamentRows(rows []dao.TournamentReward) *proto.BattleReward {
	if len(rows) == 0 {
		return nil
	}
	out := make([]*proto.PlayerReward, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, &proto.PlayerReward{
			PlayerId:         r.PlayerID,
			TemporaryAddress: r.TempAddress,
			TokenChange:      r.TokenChange,
			PointChange:      r.PointChange,
		})
	}
	return &proto.BattleReward{
		PlayerRewards: out,
		SystemFee:     0,
	}
}

// TournamentApplyMatchRewardsTx inserts tournament_rewards for this bracket match and applies user_tokens deltas.
// Idempotent: if two rows already exist for (tournament_id, round_no, match_no), reloads them and returns proto without applying wallets again.
func TournamentApplyMatchRewardsTx(tx *gorm.DB, resultType proto.GameResultType, tournamentID string, roundNo, matchNo uint32,
	p1ID int64, p1Temp string, p2ID int64, p2Temp string,
	winnerID int64, winnerTemp string, loserID int64, loserTemp string,
) (*proto.BattleReward, error) {
	p1Temp = strings.ToLower(strings.TrimSpace(p1Temp))
	p2Temp = strings.ToLower(strings.TrimSpace(p2Temp))
	winnerTemp = strings.ToLower(strings.TrimSpace(winnerTemp))
	loserTemp = strings.ToLower(strings.TrimSpace(loserTemp))

	var existing int64
	if err := tx.Model(&dao.TournamentReward{}).
		Where("tournament_id = ? AND round_no = ? AND match_no = ?", tournamentID, roundNo, matchNo).
		Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing >= 2 {
		var rows []dao.TournamentReward
		if err := tx.Where("tournament_id = ? AND round_no = ? AND match_no = ?", tournamentID, roundNo, matchNo).
			Order("player_id ASC, temp_address ASC").Find(&rows).Error; err != nil {
			return nil, err
		}
		if len(rows) < 2 {
			return nil, fmt.Errorf("tournament rewards partial rows for %s r%d m%d", tournamentID, roundNo, matchNo)
		}
		log.Debugw("tournament: match rewards idempotent (rows already exist)",
			"tournament_id", tournamentID, "round_no", roundNo, "match_no", matchNo,
			"game_result_type", resultType.String(), "existing_rows", len(rows))
		return protoBattleRewardFromTournamentRows(rows), nil
	}
	if existing == 1 {
		return nil, fmt.Errorf("tournament rewards corrupt: one row for %s r%d m%d", tournamentID, roundNo, matchNo)
	}

	var recs []dao.TournamentReward
	var rewardBranch string
	switch resultType {
	case proto.GameResultType_GAME_TIE:
		rewardBranch = "tie_both_0_10"
		recs = []dao.TournamentReward{
			{TournamentID: tournamentID, RoundNo: roundNo, MatchNo: matchNo, PlayerID: p1ID, TempAddress: p1Temp, TokenChange: tournamentRewardTieToken, PointChange: tournamentRewardTiePoint},
			{TournamentID: tournamentID, RoundNo: roundNo, MatchNo: matchNo, PlayerID: p2ID, TempAddress: p2Temp, TokenChange: tournamentRewardTieToken, PointChange: tournamentRewardTiePoint},
		}
	case proto.GameResultType_GAME_NORMAL, proto.GameResultType_GAME_KO:
		rewardBranch = "win_lose_1000_20_vs_-1000_5"
		recs = []dao.TournamentReward{
			{TournamentID: tournamentID, RoundNo: roundNo, MatchNo: matchNo, PlayerID: winnerID, TempAddress: winnerTemp, TokenChange: tournamentRewardWinToken, PointChange: tournamentRewardWinPoint},
			{TournamentID: tournamentID, RoundNo: roundNo, MatchNo: matchNo, PlayerID: loserID, TempAddress: loserTemp, TokenChange: tournamentRewardLossToken, PointChange: tournamentRewardLossPoint},
		}
	default:
		return nil, fmt.Errorf("tournament reward: unsupported game result type %v", resultType)
	}

	log.Infow("tournament: match rewards branch from room game_result_type",
		"tournament_id", tournamentID, "round_no", roundNo, "match_no", matchNo,
		"game_result_type", resultType.String(), "reward_branch", rewardBranch,
		"p1_id", p1ID, "p2_id", p2ID, "bracket_winner_id", winnerID, "bracket_loser_id", loserID)

	for i := range recs {
		if err := tx.Create(&recs[i]).Error; err != nil {
			return nil, fmt.Errorf("tournament reward insert: %w", err)
		}
		log.Infow("tournament: tournament_rewards row inserted",
			"tournament_id", tournamentID, "round_no", roundNo, "match_no", matchNo,
			"player_id", recs[i].PlayerID, "temp_address", recs[i].TempAddress,
			"token_change", recs[i].TokenChange, "point_change", recs[i].PointChange,
			"game_result_type", resultType.String())
	}

	for i := range recs {
		r := &recs[i]
		res := tx.Model(&dao.UserToken{}).Where("player_id = ?", r.PlayerID).
			Updates(map[string]any{
				"token_amount": gorm.Expr("token_amount + ?", r.TokenChange),
				"points":       gorm.Expr("points + ?", r.PointChange),
			})
		if res.Error != nil {
			return nil, fmt.Errorf("tournament reward wallet player %d: %w", r.PlayerID, res.Error)
		}
		if res.RowsAffected == 0 {
			return nil, fmt.Errorf("tournament reward: user_token missing for player_id %d", r.PlayerID)
		}
	}

	return protoBattleRewardFromTournamentRows(recs), nil
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

// TournamentOneMatchWinRewardDelta returns token/point deltas for one match win (for projecting winner totals after a hypothetical next win).
func TournamentOneMatchWinRewardDelta() (token int32, point int32) {
	return tournamentRewardWinToken, tournamentRewardWinPoint
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
