package types

import "github.com/CryptoElementals/common/rpc/proto"

// Chain tx events are internal routing events emitted by the chain worker and
// handled by the game worker. They carry the decoded proto.Transaction_* payloads.

type ChainGameCreatedTx struct {
	GameID    uint
	BlockTime int64
	Tx        *proto.Transaction_GameCreated
}

type ChainGameTurnSetupReadyTx struct {
	GameID    uint
	BlockTime int64
	Tx        *proto.Transaction_GameTurnSetupReady
}

type ChainCommitmentOnChainTx struct {
	GameID    uint
	BlockTime int64
	Tx        *proto.Transaction_CommitmentOnChain
}

type ChainCardOnChainTx struct {
	GameID    uint
	BlockTime int64
	Tx        *proto.Transaction_CardOnChain
}

