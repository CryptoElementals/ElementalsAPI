package conversion

// import (
// 	dao "github.com/CryptoElementals/common/models"
// 	"github.com/CryptoElementals/common/rpc/proto"
// 	"github.com/ethereum/go-ethereum/common/hexutil"
// )

// func GameInfoProtoToModel(gameInfoProto *proto.GameInfo) *dao.Match {
// 	var players []dao.MatchPlayer
// 	for _, player := range gameInfoProto.Players {
// 		players = append(players, dao.MatchPlayer{
// 			MatchID:       uint(gameInfoProto.RoomId),
// 			WalletAddress: player.WalletAddress,
// 			TempAddress:   player.TemporaryAddress,
// 		})
// 	}
// 	var rounds []dao.Round
// 	for _, round := range gameInfoProto.Rounds {
// 		daoRound := dao.Round{
// 			MatchID:     uint(gameInfoProto.RoomId),
// 			RoundNumber: int(round.Number),
// 			Status:      round.Status.String(),
// 		}
// 		for _, roundPlayer := range round.Players {
// 			daoRoundPlayer := dao.RoundPlayer{
// 				RoundID: uint(gameInfoProto.RoomId),
// 			}
// 			for _, roundCard := range roundPlayer.Cards {
// 				daoRoundCards := dao.RoundCard{
// 					CardCommtiment: hexutil.Encode(roundCard.SubmittedCommitment),
// 					Card: dao.Card{
// 						BaseModel: dao.BaseModel{
// 							ID: uint(roundCard.SubmittedCard.CardId),
// 						},
// 					},
// 					HealthBefore: uint32(roundCard.HealthBefore),
// 					HealthAfter:  uint32(roundCard.HealthEnd),
// 					Multiplier:   uint32(roundCard.Multiplier),
// 				}
// 				for _, item := range roundCard.SubmittedCard.Items {
// 					daoRoundCards.Items = append(daoRoundCards.Items, dao.Item{
// 						BaseModel: dao.BaseModel{
// 							ID: uint(item),
// 						},
// 					})
// 				}
// 				daoRoundPlayer.RoundCards = append(daoRoundPlayer.RoundCards, daoRoundCards)
// 			}
// 			daoRound.RoundPlayers = append(daoRound.RoundPlayers, daoRoundPlayer)
// 		}
// 		rounds = append(rounds, daoRound)
// 	}

// 	return &dao.Match{
// 		RoomContract: gameInfoProto.RoomContractAddress,
// 		Mode:         gameInfoProto.GameType.String(),
// 		Status:       uint(gameInfoProto.Status),
// 		Players:      players,
// 		Rounds:       rounds,
// 	}
// }
