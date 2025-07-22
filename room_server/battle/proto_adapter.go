package battle

import (
	pb "github.com/CryptoElementals/common/rpc/proto"
)

// ExecuteRoundProto 是 BattleEngine 的包装器，用于处理 proto 定义的输入/输出。
// 返回 RoundResult 以及在游戏结束时的 GameResult（未结束则为 nil）。
func (be *BattleEngine) ExecuteRoundProto(input *pb.RoundInput) (*pb.RoundResult, *pb.GameResult, error) {
	// 将 proto 输入转换为内部结构
	internalInput := convertProtoRoundInputToInternal(input)

	// 使用已有逻辑计算
	internalResult, err := be.ExecuteRound(internalInput)
	if err != nil {
		return nil, nil, err
	}

	// 转换为 proto 输出
	roundResult := convertInternalRoundResultToProto(internalResult)

	var gameResult *pb.GameResult
	if internalResult.IsGameOver {
		gameResult = convertInternalGameResultToProto(internalResult)
	}

	return roundResult, gameResult, nil
}

// ---------- 转换辅助函数 ----------

func convertProtoRoundInputToInternal(in *pb.RoundInput) *RoundInput {
	if in == nil {
		return nil
	}
	players := make([]PlayerRoundInput, len(in.GetPlayers()))
	mc := NewMultiplierCalculator()
	for i, p := range in.GetPlayers() {
		// 将 int32 slice 转为 int slice
		cards := make([]int, len(p.GetCards()))
		for ci, c := range p.GetCards() {
			cards[ci] = int(c)
		}
		lostHP := int(p.GetLostHP())
		players[i] = PlayerRoundInput{
			Address:    p.GetWalletAddress(),
			Cards:      cards,
			HP:         int(p.GetHP()),
			LostHP:     lostHP,
			Multiplier: mc.CalculateMultiplierByLostHP(lostHP),
		}
	}
	return &RoundInput{
		Round:   uint(in.GetRoundNumber()),
		Players: players,
	}
}

func convertInternalRoundResultToProto(in *RoundResult) *pb.RoundResult {
	if in == nil {
		return nil
	}
	players := make([]*pb.PlayerRoundStat, len(in.Players))
	for i, p := range in.Players {
		players[i] = &pb.PlayerRoundStat{
			WalletAddress: p.PlayerAddress,
			LostHP:        int32(p.LostHP),
			CardStats:     convertInternalCardStatsToProto(p.CardStats),
		}
	}
	return &pb.RoundResult{
		Players:     players,
		RoundNumber: uint32(in.Round),
		IsGameOver:  in.IsGameOver,
	}
}

func convertInternalCardStatsToProto(in []PlayerCardStat) []*pb.PlayerCardStat {
	if len(in) == 0 {
		return nil
	}
	out := make([]*pb.PlayerCardStat, len(in))
	for i, cs := range in {
		out[i] = &pb.PlayerCardStat{
			CardNumber:       int32(cs.CardNumber),
			CardID:           int32(cs.CardID),
			HPBefore:         int32(cs.HPBefore),
			HPAfter:          int32(cs.HPAfter),
			MultiplierBefore: int32(cs.MultiplierBefore),
			MultiplierAfter:  int32(cs.MultiplierAfter),
			Effects:          convertInternalEffectsToProto(cs.Effects),
			Description:      cs.Description,
			ElementRelation:  mapElementRelationStringToProto(cs.ElementRelation),
		}
	}
	return out
}

func convertInternalEffectsToProto(effects []BattleEffect) []*pb.BattleEffect {
	if len(effects) == 0 {
		return nil
	}
	out := make([]*pb.BattleEffect, len(effects))
	for i, e := range effects {
		out[i] = &pb.BattleEffect{
			Type:                mapEffectTypeStringToProto(e.Type),
			Value:               int32(e.Value),
			TargetWalletAddress: e.Target,
			Description:         e.Description,
		}
	}
	return out
}

func convertInternalGameResultToProto(in *RoundResult) *pb.GameResult {
	if in == nil {
		return nil
	}
	return &pb.GameResult{
		Multiplier:     int32(in.GameFinalMultiplier),
		GameResultType: mapGameResultTypeStringToProto(in.GameResultType),
		Reward:         convertInternalRewardToProto(in.Reward),
	}
}

func convertInternalRewardToProto(in *BattleReward) *pb.BattleReward {
	if in == nil {
		return nil
	}
	prs := make([]*pb.PlayerReward, len(in.PlayerRewards))
	for i, pr := range in.PlayerRewards {
		prs[i] = &pb.PlayerReward{
			WalletAddress: pr.PlayerAddress,
			TokenChange:   int32(pr.TokenChange),
			PointChange:   int32(pr.PointChange),
		}
	}
	return &pb.BattleReward{
		PlayerRewards: prs,
		SystemFee:     int32(in.SystemFee),
	}
}

// ---------- 枚举字符串映射 ----------

func mapEffectTypeStringToProto(s string) pb.BattleEffectType {
	switch s {
	case "attack":
		return pb.BattleEffectType_ATTACK
	case "heal":
		return pb.BattleEffectType_HEAL
	default:
		return pb.BattleEffectType_ATTACK
	}
}

func mapElementRelationStringToProto(s string) pb.ElementRelation {
	switch s {
	case "overpower":
		return pb.ElementRelation_OVER_POWER
	case "overpowered":
		return pb.ElementRelation_OVER_POWERED
	case "nurture":
		return pb.ElementRelation_NURTURE
	case "nurtured":
		return pb.ElementRelation_NURTURED
	default:
		return pb.ElementRelation_TIE
	}
}

func mapGameResultTypeStringToProto(s string) pb.GameResultType {
	switch s {
	case "ko":
		return pb.GameResultType_GAME_KO
	case "tie":
		return pb.GameResultType_GAME_TIE
	default:
		return pb.GameResultType_GAME_NORMAL
	}
}
