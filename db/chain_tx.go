package db

import (
	dao "github.com/CryptoElementals/common/models"
)

func SaveCreateRoomTx(tx *dao.CreateRoomTx) error {
	return db.Save(tx).Error
}

func SaveSetRoundReadyTx(tx *dao.SetRoundReadyTx) error {
	return db.Save(tx).Error
}

func SaveCommitmentOnChainTx(tx *dao.CommitmentOnChainTx) error {
	return db.Save(tx).Error
}

func SaveCardsOnChainTx(tx *dao.CardsOnChainTx) error {
	return db.Save(tx).Error
}

func UpdateCreateRoomTxBlockHashAndContractByGameID(gameID uint, blockHash string, contractHash string) error {
	return db.Model(&dao.CreateRoomTx{}).Where("game_id = ?", gameID).Updates(map[string]interface{}{
		"block_hash":    blockHash,
		"contract_hash": contractHash,
		"status":        dao.TxStatusSuccess,
	}).Error
}

func UpdateSetRoundReadyTxBlockHashByGameID(gameID uint, blockHash string) error {
	return db.Model(&dao.SetRoundReadyTx{}).Where("game_id = ?", gameID).Updates(map[string]interface{}{
		"block_hash": blockHash,
		"status":     dao.TxStatusSuccess,
	}).Error
}

func GetCreateRoomTx(gameID uint) (*dao.CreateRoomTx, error) {
	var tx dao.CreateRoomTx
	err := db.Where("game_id = ?", gameID).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func GetSetRoundReadyTx(gameID uint, roundNumber uint32) (*dao.SetRoundReadyTx, error) {
	var tx dao.SetRoundReadyTx
	err := db.Where("game_id = ? and round_number = ?", gameID, roundNumber).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func GetCommitmentOnChainTx(gameID uint, roundNumber uint32) (*dao.CommitmentOnChainTx, error) {
	var tx dao.CommitmentOnChainTx
	err := db.Where("game_id = ? and round_number = ?", gameID, roundNumber).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func GetCardsOnChainTx(gameID uint, roundNumber uint32) (*dao.CardsOnChainTx, error) {
	var tx dao.CardsOnChainTx
	err := db.Where("game_id = ? and round_number = ?", gameID, roundNumber).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
