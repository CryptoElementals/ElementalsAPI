package conversion

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

func DbGameInfoToProtoGameInfo(info *dao.Game) *proto.GameInfo {
	gameInfo := &proto.GameInfo{
		GameId:              uint32(info.ID),
		RoomContractAddress: info.RoomContract,
		GameType:            proto.GameType(info.Type),
		Status:              proto.GameStatus(info.Status),
		InitialHp:           int32(info.InitialHP),
	}
	// convert players
	for _, player := range info.Players {
		gameInfo.Players = append(gameInfo.Players, DbGamePlayerToProtoPlayerAddress(player))
	}
	// convert rounds
	for _, round := range info.Rounds {
		gameInfo.Rounds = append(gameInfo.Rounds, DbGameRoundToProtoGameRound(round))
	}
	// conver results
	gameResult := DbGameResultToProtoGameResult(info.GameResult)
	gameInfo.Result = gameResult
	return gameInfo
}

func DbGameResultToProtoGameResult(result *dao.GameResult) *proto.GameResult {
	gameResult := &proto.GameResult{
		Multiplier:             int32(result.Multiplier),
		WinnerWalletAddress:    result.WinnerWalletAddress,
		WinnerTemporaryAddress: result.WinnerTemporaryAddress,
		GameResultType:         result.GameResultType,
		Reward:                 DbBattleRewardToProtoBattleReward(result.BattleReward),
	}
	return gameResult
}

func DbBattleRewardToProtoBattleReward(battleReward *dao.BattleReward) *proto.BattleReward {
	return &proto.BattleReward{
		PlayerRewards: DbPlayerRewardsToProto(battleReward.PlayerRewards),
		SystemFee:     int32(battleReward.SystemFee),
	}
}

func DbPlayerRewardsToProto(playerReward []*dao.PlayerReward) []*proto.PlayerReward {
	var playerRewards []*proto.PlayerReward
	for _, playerReward := range playerReward {
		playerRewards = append(playerRewards, &proto.PlayerReward{
			WalletAddress:    playerReward.WalletAddress,
			TemporaryAddress: playerReward.TemporaryAddress,
			TokenChange:      int32(playerReward.TokenChange),
			PointChange:      int32(playerReward.PointChange),
			Offline:          playerReward.IsOffline,
		})
	}
	return playerRewards
}

func DbGamePlayerToProtoPlayerAddress(player *dao.GamePlayerInfo) *proto.PlayerAddress {
	return &proto.PlayerAddress{
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	}
}

func DbGameRoundToProtoGameRound(round *dao.Round) *proto.Round {
	return &proto.Round{
		Number:           int32(round.RoundNumber),
		Status:           proto.RoundStatus(round.Status),
		PlayerRoundInfos: DbPlayerRoundInfosToProto(round.PlayerRoundInfos),
	}
}

func DbPlayerRoundInfosToProto(playerRoundInfos []*dao.PlayerRoundInfo) []*proto.PlayerRoundInfo {
	var playerRoundInfosProto []*proto.PlayerRoundInfo
	for _, playerRoundInfo := range playerRoundInfos {
		playerRoundInfosProto = append(playerRoundInfosProto, DbPlayerRoundInfoToProto(playerRoundInfo))
	}
	return playerRoundInfosProto
}

func DbPlayerRoundInfoToProto(playerRoundInfo *dao.PlayerRoundInfo) *proto.PlayerRoundInfo {
	addr := &proto.PlayerAddress{
		WalletAddress:    playerRoundInfo.WalletAddress,
		TemporaryAddress: playerRoundInfo.TemporaryAddress,
	}
	return &proto.PlayerRoundInfo{
		PlayerAddress:       addr,
		PlayerReady:         playerRoundInfo.PlayerReady,
		SubmittedCards:      DbRoundSubmittedCardsToProto(playerRoundInfo.SubmittedCards),
		Salt:                playerRoundInfo.Salt,
		SubmittedCommitment: playerRoundInfo.SubmittedCommitment,
	}
}

func DbRoundSubmittedCardsToProto(cards []*dao.RoundSubmittedCard) []*proto.RoundSubmittedCard {
	var cardsProto []*proto.RoundSubmittedCard
	for _, card := range cards {
		cardsProto = append(cardsProto, DbRoundSubmittedCardToProto(card))
	}
	return cardsProto
}
func DbRoundSubmittedCardToProto(card *dao.RoundSubmittedCard) *proto.RoundSubmittedCard {
	return &proto.RoundSubmittedCard{
		PlayerHealthBefore: card.HealthBefore,
		PlayerHealthEnd:    card.HealthAfter,
		MultiplierBefore:   card.MultiplierBefore,
		MultiplierAfter:    card.MultiplierAfter,
		Description:        card.Description,
		ElementRelation:    card.ElementRelation,
		Effects:            DbCardEffectsToProto(card.CardEffects),
		SubmittedCardId:    uint32(card.CardID),
	}
}

func DbCardEffectsToProto(effects []*dao.CardEffect) []*proto.BattleEffect {
	var effectsProto []*proto.BattleEffect
	for _, effect := range effects {
		effectsProto = append(effectsProto, DbCardEffectToProto(effect))
	}
	return effectsProto
}

func DbCardEffectToProto(effect *dao.CardEffect) *proto.BattleEffect {
	return &proto.BattleEffect{
		Type:                   effect.Type,
		Value:                  effect.Value,
		Description:            effect.Description,
		TargetWalletAddress:    effect.TargetWalletAddress,
		TargetTemporaryAddress: effect.TargetTemporaryAddress,
	}
}

func DbRoundToRoundResult(round *dao.Round) *proto.RoundResult {
	return &proto.RoundResult{
		Players:     DbPlayerRoundInfosToProtoPlayerRoundStats(round.PlayerRoundInfos),
		RoundNumber: round.RoundNumber,
		IsGameOver:  round.IsLastRound,
	}
}

func DbPlayerRoundInfosToProtoPlayerRoundStats(playerRoundInfo []*dao.PlayerRoundInfo) []*proto.PlayerRoundStat {
	var playerRoundStats []*proto.PlayerRoundStat
	for _, playerRoundInfo := range playerRoundInfo {
		playerRoundStats = append(playerRoundStats, DbPlayerRoundInfoToProtoPlayerRoundStat(playerRoundInfo))
	}
	return playerRoundStats
}

func DbPlayerRoundInfoToProtoPlayerRoundStat(playerRoundInfo *dao.PlayerRoundInfo) *proto.PlayerRoundStat {
	return &proto.PlayerRoundStat{
		WalletAddress:    playerRoundInfo.WalletAddress,
		TemporaryAddress: playerRoundInfo.TemporaryAddress,
		LostHP:           playerRoundInfo.LostHP,
		CardStats:        DbRoundSubmittedCardToProtoPlayerCardStats(playerRoundInfo.SubmittedCards),
	}
}

func DbRoundSubmittedCardToProtoPlayerCardStats(roundSubmittedCard []*dao.RoundSubmittedCard) []*proto.PlayerCardStat {
	var playerCardStats []*proto.PlayerCardStat
	for i, card := range roundSubmittedCard {
		playerCardStats = append(playerCardStats, DbRoundSubmittedCardToProtoPlayerCardStat(i, card))
	}
	return playerCardStats
}

func DbRoundSubmittedCardToProtoPlayerCardStat(i int, card *dao.RoundSubmittedCard) *proto.PlayerCardStat {
	return &proto.PlayerCardStat{
		CardNumber:       int32(i),
		CardID:           int32(card.CardID),
		HPBefore:         int32(card.HealthBefore),
		HPAfter:          int32(card.HealthAfter),
		MultiplierBefore: int32(card.MultiplierBefore),
		MultiplierAfter:  int32(card.MultiplierAfter),
		Description:      card.Description,
		ElementRelation:  card.ElementRelation,
		Effects:          DbCardEffectsToProto(card.CardEffects),
	}
}

func DbGameToProtoGamePhase(game *dao.Game, currentRound *dao.Round) *proto.GamePhase {
	gamePhase := &proto.GamePhase{
		GameType: proto.GameType(game.Type),
	}

	for _, playerInfo := range currentRound.PlayerRoundInfos {
		addr := types.NewPlayerAddress(
			playerInfo.WalletAddress,
			playerInfo.TemporaryAddress,
		).ToProto()
		gamePhase.Players = append(gamePhase.Players, &proto.GamePhasePlayer{
			Address:     addr,
			IsConfirmed: playerInfo.PlayerReady,
		})
	}
	gamePhase.PvPInfo = &proto.PvPInfo{
		GameID:          uint32(game.ID),
		Status:          proto.PlayerStatus(game.Status),
		ContractAddress: game.RoomContract,
		BeginAt:         uint64(game.CreatedAt.Unix()),
		TimeoutDuration: uint64(game.RoundTimeout),
	}
	return gamePhase
}
