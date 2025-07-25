package battle

import (
	"strings"

	"github.com/CryptoElementals/common/config"
)

// RewardCalculator reward calculator
type RewardCalculator struct {
	BaseStake int
}

// NewRewardCalculator create a new reward calculator
func NewRewardCalculator() *RewardCalculator {
	return &RewardCalculator{BaseStake: config.GameParams.BaseStake}
}

// CalculateRewards calculate battle rewards
func (rc *RewardCalculator) CalculateRewards(result *RoundResult) BattleReward {
	// 如果游戏尚未结束或 GameResult 为空，直接返回 nil
	if result == nil || result.GameResult == nil {
		return BattleReward{}
	}

	gr := result.GameResult
	baseStake := rc.BaseStake
	finalMultiplier := float64(gr.Multiplier)
	var playerRewards []PlayerReward
	var systemFee int

	switch gr.GameResultType {
	case GAME_TIE:
		// 平局情况：每个人扣除basetoken×0.008，积分basetoken×0.008，系统抽水是所有人扣除的token之和
		totalDeducted := 0
		for _, player := range result.Players {
			tokenDeduction := int(float64(baseStake) * 0.008)
			pointGain := int(float64(baseStake) * 0.008)
			totalDeducted += tokenDeduction

			playerRewards = append(playerRewards, PlayerReward{
				WalletAddress:    player.WalletAddress,
				TemporaryAddress: player.TemporaryAddress,
				TokenChange:      -tokenDeduction,
				PointChange:      pointGain,
			})
		}
		systemFee = totalDeducted

	case GAME_NORMAL, GAME_KO:
		if gr.WinnerWalletAddress != "" {
			// 解析赢家地址列表
			winnerAddresses := make(map[string]bool)
			winnerList := []string{gr.WinnerWalletAddress}
			if gr.WinnerWalletAddress != "" {
				winnerList = strings.Split(gr.WinnerWalletAddress, "|")
				for _, addr := range winnerList {
					winnerAddresses[addr] = true
				}
			}

			winnerCount := len(winnerList)
			loserCount := len(result.Players) - winnerCount
			totalPool := int(float64(baseStake) * finalMultiplier)

			// 赢家平分总奖池×(1-0.016)
			winnerTokenPerPlayer := int(float64(totalPool)*(1.0-0.016)) / winnerCount

			// 输家平均承担扣除总奖池
			loserTokenPerPlayer := totalPool / loserCount

			// 积分计算
			var winnerPointPerPlayer, loserPointPerPlayer int
			if gr.GameResultType == GAME_NORMAL {
				// NORMAL: 赢家平分总奖池×0.012，输家平分总奖池×0.004
				winnerPointPerPlayer = int(float64(totalPool)*0.012) / winnerCount
				loserPointPerPlayer = int(float64(totalPool)*0.004) / loserCount
			} else {
				// KO: 赢家平分总奖池×0.016，输家没有积分收益
				winnerPointPerPlayer = int(float64(totalPool)*0.016) / winnerCount
				loserPointPerPlayer = 0
			}

			// 分配奖励
			for _, player := range result.Players {
				if winnerAddresses[player.WalletAddress] {
					// 赢家
					playerRewards = append(playerRewards, PlayerReward{
						WalletAddress:    player.WalletAddress,
						TemporaryAddress: player.TemporaryAddress,
						TokenChange:      winnerTokenPerPlayer,
						PointChange:      winnerPointPerPlayer,
					})
				} else {
					// 输家
					playerRewards = append(playerRewards, PlayerReward{
						WalletAddress:    player.WalletAddress,
						TemporaryAddress: player.TemporaryAddress,
						TokenChange:      -loserTokenPerPlayer,
						PointChange:      loserPointPerPlayer,
					})
				}
			}

			systemFee = int(float64(totalPool) * 0.016)
		}
	}

	return BattleReward{
		PlayerRewards: playerRewards,
		SystemFee:     systemFee,
	}
}
