package conversion

import (
	"sort"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

func DbGameInfoToProtoGameInfo(info *dao.Game) *proto.GameInfo {
	if info == nil {
		return nil
	}
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
	if result == nil {
		return nil
	}
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
	if battleReward == nil {
		return nil
	}
	return &proto.BattleReward{
		PlayerRewards: DbPlayerRewardsToProto(battleReward.PlayerRewards),
		SystemFee:     int32(battleReward.SystemFee),
	}
}

func DbPlayerRewardsToProto(playerReward []*dao.PlayerReward) []*proto.PlayerReward {
	if len(playerReward) == 0 {
		return nil
	}
	var playerRewards []*proto.PlayerReward
	for _, playerReward := range playerReward {
		playerRewards = append(playerRewards, &proto.PlayerReward{
			WalletAddress:          playerReward.WalletAddress,
			TemporaryAddress:       playerReward.TemporaryAddress,
			TokenChange:            int32(playerReward.TokenChange),
			PointChange:            int32(playerReward.PointChange),
			Offline:                playerReward.IsOffline,
			Surrendered:            playerReward.Surrendered,
			PlayerGameResultStatus: playerReward.PlayerGameResultStatus,
		})
	}
	return playerRewards
}

func DbGamePlayerToProtoPlayerAddress(player *dao.GamePlayerInfo) *proto.PlayerAddress {
	if player == nil {
		return nil
	}
	return &proto.PlayerAddress{
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	}
}

func DbGameRoundToProtoGameRound(round *dao.Round) *proto.Round {
	if round == nil {
		return nil
	}
	return &proto.Round{
		Number:           int32(round.RoundNumber),
		Status:           proto.RoundStatus(round.Status),
		PlayerRoundInfos: DbPlayerRoundInfosToProto(round.PlayerRoundInfos),
	}
}

func DbPlayerRoundInfosToProto(playerRoundInfos []*dao.PlayerRoundInfo) []*proto.PlayerRoundInfo {
	if len(playerRoundInfos) == 0 {
		return nil
	}
	var playerRoundInfosProto []*proto.PlayerRoundInfo
	for _, playerRoundInfo := range playerRoundInfos {
		playerRoundInfosProto = append(playerRoundInfosProto, DbPlayerRoundInfoToProto(playerRoundInfo))
	}
	return playerRoundInfosProto
}

func DbPlayerRoundInfoToProto(playerRoundInfo *dao.PlayerRoundInfo) *proto.PlayerRoundInfo {
	if playerRoundInfo == nil {
		return nil
	}
	addr := &proto.PlayerAddress{
		WalletAddress:    playerRoundInfo.WalletAddress,
		TemporaryAddress: playerRoundInfo.TemporaryAddress,
	}
	// Note: Salt and SubmittedCommitment are now in RoundSubmittedCard, not PlayerRoundInfo
	return &proto.PlayerRoundInfo{
		PlayerAddress:  addr,
		PlayerReady:    playerRoundInfo.PlayerReady,
		SubmittedCards: DbRoundSubmittedCardsToProto(playerRoundInfo.SubmittedCards),
	}
}

func DbRoundSubmittedCardsToProto(cards []*dao.RoundSubmittedCard) []*proto.RoundSubmittedCard {
	if len(cards) == 0 {
		return nil
	}
	var cardsProto []*proto.RoundSubmittedCard
	for _, card := range cards {
		cardsProto = append(cardsProto, DbRoundSubmittedCardToProto(card))
	}
	return cardsProto
}
func DbRoundSubmittedCardToProto(card *dao.RoundSubmittedCard) *proto.RoundSubmittedCard {
	if card == nil {
		return nil
	}
	return &proto.RoundSubmittedCard{
		PlayerHealthBefore:  card.HealthBefore,
		PlayerHealthEnd:     card.HealthAfter,
		MultiplierBefore:    card.MultiplierBefore,
		MultiplierAfter:     card.MultiplierAfter,
		Description:         card.Description,
		ElementRelation:     card.ElementRelation,
		Effects:             DbCardEffectsToProto(card.CardEffects),
		SubmittedCardId:     uint32(card.CardID),
		SubmittedCommitment: card.SubmittedCommitment,
		Salt:                card.Salt,
	}
}

func DbCardEffectsToProto(effects []*dao.CardEffect) []*proto.BattleEffect {
	if len(effects) == 0 {
		return nil
	}
	var effectsProto []*proto.BattleEffect
	for _, effect := range effects {
		effectsProto = append(effectsProto, DbCardEffectToProto(effect))
	}
	return effectsProto
}

func DbCardEffectToProto(effect *dao.CardEffect) *proto.BattleEffect {
	if effect == nil {
		return nil
	}
	return &proto.BattleEffect{
		Type:                   effect.Type,
		Value:                  effect.Value,
		Description:            effect.Description,
		TargetWalletAddress:    effect.TargetWalletAddress,
		TargetTemporaryAddress: effect.TargetTemporaryAddress,
	}
}

func DbRoundToRoundResult(round *dao.Round) *proto.RoundResult {
	if round == nil {
		return nil
	}
	return &proto.RoundResult{
		Players:      DbPlayerRoundInfosToProtoPlayerRoundStats(round.PlayerRoundInfos),
		RoundNumber:  round.RoundNumber,
		IsGameOver:   round.IsLastRound,
		RoundEndTime: uint64(round.RoundEndTime),
	}
}

func DbPlayerRoundInfosToProtoPlayerRoundStats(playerRoundInfo []*dao.PlayerRoundInfo) []*proto.PlayerRoundStat {
	if len(playerRoundInfo) == 0 {
		return nil
	}
	var playerRoundStats []*proto.PlayerRoundStat
	for _, playerRoundInfo := range playerRoundInfo {
		playerRoundStats = append(playerRoundStats, DbPlayerRoundInfoToProtoPlayerRoundStat(playerRoundInfo))
	}
	return playerRoundStats
}

func DbPlayerRoundInfoToProtoPlayerRoundStat(playerRoundInfo *dao.PlayerRoundInfo) *proto.PlayerRoundStat {
	if playerRoundInfo == nil {
		return nil
	}
	return &proto.PlayerRoundStat{
		WalletAddress:    playerRoundInfo.WalletAddress,
		TemporaryAddress: playerRoundInfo.TemporaryAddress,
		LostHP:           playerRoundInfo.LostHP,
		CardStats:        DbRoundSubmittedCardToProtoPlayerCardStats(playerRoundInfo.SubmittedCards),
	}
}

func DbRoundSubmittedCardToProtoPlayerCardStats(roundSubmittedCard []*dao.RoundSubmittedCard) []*proto.PlayerCardStat {
	if len(roundSubmittedCard) == 0 {
		return nil
	}
	var playerCardStats []*proto.PlayerCardStat
	for i, card := range roundSubmittedCard {
		playerCardStats = append(playerCardStats, DbRoundSubmittedCardToProtoPlayerCardStat(i, card))
	}
	return playerCardStats
}

func DbRoundSubmittedCardToProtoPlayerCardStat(i int, card *dao.RoundSubmittedCard) *proto.PlayerCardStat {
	if card == nil {
		return nil
	}
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
	if game == nil || currentRound == nil {
		return nil
	}
	gamePhase := &proto.GamePhase{
		GameType: proto.GameType(game.Type),
	}

	for _, playerInfo := range currentRound.PlayerRoundInfos {
		addr := types.NewPlayerAddress(
			playerInfo.WalletAddress,
			playerInfo.TemporaryAddress,
		).ToProto()
		cards := make([]uint32, 0, len(playerInfo.SubmittedCards))
		sort.Slice(playerInfo.SubmittedCards, func(i, j int) bool {
			return playerInfo.SubmittedCards[i].CardNumber < playerInfo.SubmittedCards[j].CardNumber
		})

		for _, sc := range playerInfo.SubmittedCards {
			cards = append(cards, uint32(sc.CardID))
		}

		// Get commitment from first card (if exists)
		var commitment []byte
		if len(playerInfo.SubmittedCards) > 0 {
			commitment = playerInfo.SubmittedCards[0].SubmittedCommitment
		}

		gamePhase.Players = append(gamePhase.Players, &proto.GamePhasePlayer{
			Address:     addr,
			IsConfirmed: playerInfo.PlayerReady,
			Commitment:  commitment,
			Cards:       cards,
		})
	}
	playerStatus := proto.PlayerStatus(0)
	timeoutDuration := int64(0)
	beginAt := uint64(currentRound.SetupOnChainAt)
	switch game.Status {
	case proto.GameStatus_GAME_INIT:
		playerStatus = proto.PlayerStatus_PLAYER_MATCHED
		// game waitting confirmed for the first round
		timeoutDuration = game.GameArgs.GameMatchTimeout
	case proto.GameStatus_GAME_RUNNING:
		playerStatus = proto.PlayerStatus_PLAYER_IN_GAME
		switch currentRound.Status {
		case proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION,
			proto.RoundStatus_ROUND_COMPLETED,
			proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN:
			// waitting for confimation
			timeoutDuration = game.GameArgs.RoundConfirmTimeout
		case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS, proto.RoundStatus_ROUND_WAITTING_CARDS:
			// round submitting cards
			timeoutDuration = game.GameArgs.RoundTimeout
		}
	case proto.GameStatus_GAME_END:
		playerStatus = proto.PlayerStatus_PLAYER_UNKNOWN
		// round continue
		timeoutDuration = game.ContinueTimeout
		beginAt = uint64(game.UpdatedAt.Unix())
	}

	if beginAt == 0 {
		beginAt = uint64(currentRound.CreatedAt.Unix())
	}

	gamePhase.PvPInfo = &proto.PvPInfo{
		GameID:          uint32(game.ID),
		Status:          playerStatus,
		ContractAddress: game.RoomContract,
		BeginAt:         beginAt,
		TimeoutDuration: uint64(timeoutDuration),
		RoundNumber:     uint64(currentRound.RoundNumber),
	}
	return gamePhase
}
