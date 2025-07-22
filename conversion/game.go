package conversion

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

func DbGameInfoToProtoGameInfo(info *dao.Game) *proto.GameInfo {
	gameInfo := &proto.GameInfo{
		GameId:              uint32(info.ID),
		RoomContractAddress: info.RoomContract,
		GameType:            proto.GameType(info.Type),
		Status:              proto.GameStatus(info.Status),
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

	return gameInfo
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
