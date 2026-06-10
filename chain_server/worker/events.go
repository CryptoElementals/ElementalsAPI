package worker

import "github.com/CryptoElementals/common/rpc/proto"

type RequireGameCreationEvent struct {
	GameID         int64
	InitialHP      int64
	RoundTimeout   int64
	MaxRoundNumber int64
	TournamentID   int64
	TierNo         int64
	Players        []PlayerAddress
}

type RequireSetupNewTurnEvent struct {
	GameID      int64
	RoundNumber uint32
	TurnNumber  uint32
}

type RoomContractTask struct {
	Index uint8
	Task  []byte
}

type PlayerAddress struct {
	Id               int64
	TemporaryAddress string
}

func PlayerAddressFromProto(addr *proto.PlayerAddress) PlayerAddress {
	if addr == nil {
		return PlayerAddress{}
	}
	return PlayerAddress{Id: addr.GetId(), TemporaryAddress: addr.GetTemporaryAddress()}
}

func RequireGameCreationFromProto(evt *proto.RequireGameCreationEvent) *RequireGameCreationEvent {
	if evt == nil {
		return nil
	}
	out := &RequireGameCreationEvent{
		GameID: evt.GetGameId(), InitialHP: evt.GetInitialHp(),
		RoundTimeout: evt.GetRoundTimeout(), MaxRoundNumber: evt.GetMaxRoundNumber(),
		TournamentID: evt.GetTournamentId(), TierNo: evt.GetTierNo(),
	}
	for _, p := range evt.GetPlayers() {
		out.Players = append(out.Players, PlayerAddressFromProto(p))
	}
	return out
}

func RequireSetupNewTurnFromProto(evt *proto.RequireSetupNewTurnEvent) *RequireSetupNewTurnEvent {
	if evt == nil {
		return nil
	}
	return &RequireSetupNewTurnEvent{
		GameID: evt.GetGameId(), RoundNumber: evt.GetRoundNumber(), TurnNumber: evt.GetTurnNumber(),
	}
}
