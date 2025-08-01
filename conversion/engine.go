package conversion

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

func DbRoundToProtoRoundInput(dbRound *dao.Round) *proto.RoundInput {
	if dbRound == nil {
		return nil
	}
	return &proto.RoundInput{
		RoundNumber: int32(dbRound.RoundNumber),
		Players:     DbPlayerRoundInfoToProtoPlayerRoundInput(dbRound.PlayerRoundInfos),
	}
}

func DbPlayerRoundInfoToProtoPlayerRoundInput(playerRoundInfo []*dao.PlayerRoundInfo) []*proto.PlayerRoundInput {
	if len(playerRoundInfo) == 0 {
		return nil
	}
	playerRoundInput := make([]*proto.PlayerRoundInput, 0, len(playerRoundInfo))
	for _, p := range playerRoundInfo {
		cards := make([]int32, 0, len(p.SubmittedCards))
		for _, c := range p.SubmittedCards {
			cards = append(cards, int32(c.CardID))
		}
		playerRoundInput = append(playerRoundInput, &proto.PlayerRoundInput{
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
			Commitment:       p.SubmittedCommitment,
			Cards:            cards,
			Surrendered:      p.Surrendered,
		})
	}
	return playerRoundInput
}

func ProtoBattleEffectsToDbCardEffects(protoCardEffects []*proto.BattleEffect) []*dao.CardEffect {
	if len(protoCardEffects) == 0 {
		return nil
	}
	dbCardEffects := make([]*dao.CardEffect, 0, len(protoCardEffects))
	for _, effect := range protoCardEffects {
		dbCardEffects = append(dbCardEffects, &dao.CardEffect{
			Type:                   effect.Type,
			Value:                  effect.Value,
			Description:            effect.Description,
			TargetWalletAddress:    effect.TargetWalletAddress,
			TargetTemporaryAddress: effect.TargetTemporaryAddress,
		})
	}
	return dbCardEffects
}

func ProtoGameResultToDbGameResult(protoGameResult *proto.GameResult) *dao.GameResult {
	if protoGameResult == nil {
		return nil
	}
	return &dao.GameResult{
		Multiplier:             protoGameResult.Multiplier,
		WinnerWalletAddress:    protoGameResult.WinnerWalletAddress,
		WinnerTemporaryAddress: protoGameResult.WinnerTemporaryAddress,
		GameResultType:         protoGameResult.GameResultType,
		BattleReward:           ProtoBattleRewardsToDbBattleReward(protoGameResult.Reward),
	}
}

func ProtoBattleRewardsToDbBattleReward(protoBattleReward *proto.BattleReward) *dao.BattleReward {
	if protoBattleReward == nil {
		return nil
	}
	return &dao.BattleReward{
		SystemFee:     protoBattleReward.SystemFee,
		PlayerRewards: ProtoPlayerRewardsToDbPlayerRewards(protoBattleReward.PlayerRewards),
	}
}

func ProtoPlayerRewardsToDbPlayerRewards(protoPlayerRewards []*proto.PlayerReward) []*dao.PlayerReward {
	if len(protoPlayerRewards) == 0 {
		return nil
	}
	dbPlayerRewards := make([]*dao.PlayerReward, 0, len(protoPlayerRewards))
	for _, p := range protoPlayerRewards {
		dbPlayerRewards = append(dbPlayerRewards, &dao.PlayerReward{
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
			TokenChange:      p.TokenChange,
			PointChange:      p.PointChange,
			IsOffline:        p.Offline,
			Surrendered:      p.Surrendered,
		})
	}
	return dbPlayerRewards
}
