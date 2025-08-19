package battle

import pb "github.com/CryptoElementals/common/rpc/proto"

// ExecuteRoundProto 提供 BattleEngine 的 proto 友好包装，
// 返回 proto.RoundResult，以及游戏结束时的 proto.GameResult（未结束则为 nil）。
func (be *BattleEngine) ExecuteRoundProto(input *pb.RoundInput) (*pb.RoundResult, *pb.GameResult, error) {
	internalInput := convertProtoRoundInputToInternal(input)
	internalResult, err := be.ExecuteRound(internalInput)
	if err != nil {
		return nil, nil, err
	}

	roundResult := convertInternalRoundResultToProto(internalResult)
	var gameResult *pb.GameResult
	if internalResult.IsGameOver {
		gameResult = convertInternalGameResultToProto(internalResult)
	}
	return roundResult, gameResult, nil
}

// -------------------- proto -> internal --------------------

func convertProtoRoundInputToInternal(in *pb.RoundInput) *RoundInput {
	if in == nil {
		return nil
	}
	players := make([]PlayerRoundInput, len(in.GetPlayers()))
	for i, p := range in.GetPlayers() {
		cards := make([]int, len(p.GetCards()))
		for j, c := range p.GetCards() {
			cards[j] = int(c)
		}
		players[i] = PlayerRoundInput{
			WalletAddress:    p.GetWalletAddress(),
			TemporaryAddress: p.GetTemporaryAddress(),
			Cards:            cards,
			HP:               int(p.GetHP()),
			LostHP:           int(p.GetLostHP()),
			Commitment:       p.GetCommitment(),
			Surrendered:      p.GetSurrendered(),
			// Status 字段将在 ValidateRoundInput 中设置默认值
		}
	}
	return &RoundInput{
		RoundNumber: uint32(in.GetRoundNumber()),
		Players:     players,
		Reason:      in.GetReason(),
	}
}

// -------------------- internal -> proto --------------------

func convertInternalRoundResultToProto(in *RoundResult) *pb.RoundResult {
	if in == nil {
		return nil
	}
	prs := make([]*pb.PlayerRoundStat, len(in.Players))
	for i, p := range in.Players {
		prs[i] = &pb.PlayerRoundStat{
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
			LostHP:           int32(p.LostHP),
			CardStats:        convertInternalCardStatsToProto(p.CardStats),
		}
	}
	return &pb.RoundResult{
		Players:     prs,
		RoundNumber: in.RoundNumber,
		IsGameOver:  in.IsGameOver,
	}
}

func convertInternalCardStatsToProto(in []PlayerCardStat) []*pb.PlayerCardStat {
	if len(in) == 0 {
		return nil
	}
	out := make([]*pb.PlayerCardStat, len(in))
	for i, c := range in {
		out[i] = &pb.PlayerCardStat{
			CardNumber:       int32(c.CardNumber),
			CardID:           int32(c.CardID),
			HPBefore:         int32(c.HPBefore),
			HPAfter:          int32(c.HPAfter),
			MultiplierBefore: int32(c.MultiplierBefore),
			MultiplierAfter:  int32(c.MultiplierAfter),
			Effects:          convertInternalEffectsToProto(c.Effects),
			Description:      c.Description,
			ElementRelation:  pb.ElementRelation(c.ElementRelation),
		}
	}
	return out
}

func convertInternalEffectsToProto(effs []BattleEffect) []*pb.BattleEffect {
	if len(effs) == 0 {
		return nil
	}
	out := make([]*pb.BattleEffect, len(effs))
	for i, e := range effs {
		out[i] = &pb.BattleEffect{
			Type:                   pb.BattleEffectType(e.Type),
			Value:                  int32(e.Value),
			TargetWalletAddress:    e.TargetWalletAddress,
			TargetTemporaryAddress: e.TargetTemporaryAddress,
			Description:            e.Description,
		}
	}
	return out
}

func convertInternalGameResultToProto(in *RoundResult) *pb.GameResult {
	if in == nil || in.GameResult == nil {
		return nil
	}
	gr := in.GameResult
	return &pb.GameResult{
		Multiplier:             int32(gr.Multiplier),
		WinnerWalletAddress:    gr.WinnerWalletAddress,
		WinnerTemporaryAddress: gr.WinnerTemporaryAddress,
		GameResultType:         pb.GameResultType(gr.GameResultType),
		Reward:                 convertInternalRewardToProto(&gr.Reward),
	}
}

func convertInternalRewardToProto(in *BattleReward) *pb.BattleReward {
	if in == nil {
		return nil
	}
	players := make([]*pb.PlayerReward, len(in.PlayerRewards))
	for i, pr := range in.PlayerRewards {
		players[i] = &pb.PlayerReward{
			WalletAddress:          pr.WalletAddress,
			TemporaryAddress:       pr.TemporaryAddress,
			TokenChange:            int32(pr.TokenChange),
			PointChange:            int32(pr.PointChange),
			Offline:                pr.IsOffline,
			Surrendered:            pr.IsSurrendered,
			PlayerGameResultStatus: pr.PlayerGameResultStatus,
		}
	}
	return &pb.BattleReward{
		PlayerRewards: players,
		SystemFee:     int32(in.SystemFee),
	}
}
