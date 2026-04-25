package dao

// Chain tx pool item kinds; order matches flush priority (create → set turn → commitment → card).
const (
	ChainTxPoolKindCreateRoom   uint8 = 1
	ChainTxPoolKindSetTurnReady uint8 = 2
	ChainTxPoolKindCommitment   uint8 = 3
	ChainTxPoolKindCard         uint8 = 4
)

// ChainTxPoolItem is a pending room contract operation queued for batch submission to an L2.
// Natural key: (chain_id, kind, game_id, player_temporary_addr, round_number, turn_number).
// For create_room and set_turn_ready, player_temporary_addr is "" and round/turn use 0 for unused fields
// (set_turn still sets round/turn; create_room uses 0,0 for both).
type ChainTxPoolItem struct {
	BaseModel

	ChainID int64 `gorm:"not null" json:"chain_id"`
	GameID  int64 `gorm:"not null;uniqueIndex:ux_chain_tx_pool_natural" json:"game_id"`
	Kind    uint8 `gorm:"not null;uniqueIndex:ux_chain_tx_pool_natural" json:"kind"`
	// PlayerTemporaryAddr is empty for create_room; used for commitment/card matching event key.
	PlayerTemporaryAddr string `gorm:"size:64;not null;default:'';uniqueIndex:ux_chain_tx_pool_natural" json:"player_temporary_addr"`
	RoundNumber         uint32 `gorm:"not null;uniqueIndex:ux_chain_tx_pool_natural" json:"round_number"`
	TurnNumber          uint32 `gorm:"not null;uniqueIndex:ux_chain_tx_pool_natural" json:"turn_number"`
	// Opaque: proto.Marshal for kind 3/4; json.Marshal for 1/2 in chain package.
	Payload []byte `gorm:"type:mediumblob;not null" json:"-"`
}

func (ChainTxPoolItem) TableName() string { return "chain_tx_pool_items" }
