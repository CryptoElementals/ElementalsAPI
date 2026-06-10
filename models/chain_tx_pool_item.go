package dao

import "time"

const (
	ChainTxPoolKindCreateRoom   uint8 = 1
	ChainTxPoolKindSetTurnReady uint8 = 2
	ChainTxPoolKindCommitment   uint8 = 3
	ChainTxPoolKindCard         uint8 = 4
)

type ChainTxPoolItem struct {
	BaseModel

	ChainID int64 `gorm:"not null;index:idx_chain_tx_pool_drain,priority:1" json:"chain_id"`
	GameID  int64 `gorm:"not null;index:idx_chain_tx_pool_game_id" json:"game_id"`
	Kind    uint8 `gorm:"not null" json:"kind"`
	PlayerTemporaryAddr string `gorm:"size:64;not null;default:''" json:"player_temporary_addr"`
	RoundNumber         uint32 `gorm:"not null" json:"round_number"`
	TurnNumber          uint32 `gorm:"not null" json:"turn_number"`
	Payload             []byte `gorm:"type:mediumblob;not null" json:"-"`
	ClaimedAt           *time.Time `gorm:"index:idx_chain_tx_pool_drain,priority:2" json:"claimed_at"`
}

func (ChainTxPoolItem) TableName() string { return "chain_tx_pool_items" }
