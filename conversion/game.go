package conversion

import (
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

func DbGameInfoToProtoGameInfo(info *dao.GameInfo) *proto.GameInfo {
	gameInfo := &proto.GameInfo{
		GameId:              uint32(info.ID),
		RoomContractAddress: info.RoomContract,
		GameType:            proto.GameType(info.Type),
		Status:              proto.GameStatus(info.Status),
	}
	// convert players
	for _, player := range info.Players {
		gameInfo.Players = append(gameInfo.Players, DbGamePlayerToProtoPlayerAddress(&player))
	}
	// convert rounds
	for _, round := range info.Rounds {
		gameInfo.Rounds = append(gameInfo.Rounds, DbGameRoundToProtoGameRound(&round))
	}
	// conver results
	return gameInfo
}

func DbGamePlayerToProtoPlayerAddress(player *dao.GamePlayer) *proto.PlayerAddress {
	return &proto.PlayerAddress{
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	}
}

func DbGameRoundToProtoGameRound(round *dao.Round) *proto.Round {
	return &proto.Round{
		Number:  int32(round.RoundNumber),
		Status:  proto.RoundStatus(round.Status),
		Players: DbPlayerRoundInfosToProto(round.PlayerRoundInfos),
	}
}

func DbPlayerRoundInfosToProto(playerRoundInfos []dao.PlayerRoundInfo) []*proto.PlayerRoundInfo {
	var playerRoundInfosProto []*proto.PlayerRoundInfo
	for _, playerRoundInfo := range playerRoundInfos {
		playerRoundInfosProto = append(playerRoundInfosProto, DbPlayerRoundInfoToProto(&playerRoundInfo))
	}
	return playerRoundInfosProto
}

func DbPlayerRoundInfoToProto(playerRoundInfo *dao.PlayerRoundInfo) *proto.PlayerRoundInfo {
	return &proto.PlayerRoundInfo{
		PlayerAddress:       DbGamePlayerToProtoPlayerAddress(&playerRoundInfo.GamePlayer),
		Cards:               DbRoundSubmittedCardsToProto(playerRoundInfo.RoundSubmittedCards),
		Salt:                playerRoundInfo.Salt,
		SubmittedCommitment: playerRoundInfo.SubmittedCommitment,
	}
}

func DbRoundSubmittedCardsToProto(cards []dao.RoundSubmittedCard) []*proto.RoundSubmittedCard {
	var cardsProto []*proto.RoundSubmittedCard
	for _, card := range cards {
		cardsProto = append(cardsProto, DbRoundSubmittedCardToProto(&card))
	}
	return cardsProto
}
func DbRoundSubmittedCardToProto(card *dao.RoundSubmittedCard) *proto.RoundSubmittedCard {
	return &proto.RoundSubmittedCard{
		PlayerHealthBefore: card.HealthBefore,
		PlayerHealthEnd:    card.HealthAfter,
		Multiplier:         card.Multiplier,
		SubmittedCardId:    card.CardID,
	}
}
