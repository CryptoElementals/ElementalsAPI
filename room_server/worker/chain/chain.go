package chain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type batchTxEvent struct {
	txs       *proto.TransactionBatch
	blockNum  uint64
	blockHash []byte
}

type Chain struct {
	ctx                  context.Context
	workerManager        *worker.WorkerManager
	createRoomTxToGameID cache.Cache
	gameContractToRoomID cache.Cache
	roomMgrClient        *concurrentRoomClient
}

func NewChain(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	chainID int64,
	client bind.ContractBackend,
	roomManagerContractAddressHex string,
	wallets []*wallet.Wallet,
	dataCache cache.Cache,
	isDevelop ...bool,
) (*Chain, error) {
	roomMgrCli, err := newConcurrentRoomClient(ctx, client, roomManagerContractAddressHex, wallets, chainID)
	if err != nil {
		log.Errorf("newConcurrentRoomManagerClient: create room manager client failed: %s", err.Error())
		return nil, err
	}
	return &Chain{
		ctx:                  ctx,
		workerManager:        workerManager,
		createRoomTxToGameID: cache.WithPrefix("create_room_tx_to_game_id", dataCache),
		gameContractToRoomID: cache.WithPrefix("game_contract_to_room_id", dataCache),
		roomMgrClient:        roomMgrCli,
	}, nil
}

func (c *Chain) Start() error {
	txs, err := db.ListCreateRoomTxWithNoContractAddr()
	if err != nil {
		return err
	}
	for _, tx := range txs {
		err = c.createRoomTxToGameID.Set(tx.TxHash, fmt.Sprint(tx.GameID), int(time.Hour.Seconds()))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Chain) CreateRoomContract(evt *types.RequireContractCreationEvent) error {
	err := c.createRoomContract(evt.GameID, evt.Players, evt.InitialHP, evt.RoundTimeout, evt.MaxRoundNumber)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chain) SetRoundReady(evt *types.RequireSetupNewRoundEvent) error {
	err := c.setRoundReady(evt.GameID, evt.RoundNumber, evt.ContractAddress)
	if err != nil {
		return err
	}
	return nil
}

func (c *Chain) batchSendTxs(evt *batchTxEvent) {
	c.handleChainEvents(evt)
}

func (c *Chain) createRoomContract(gameID uint, players []types.PlayerAddress, initialHP int64, roundTimeout int64, maxRounds int64) error {
	player1WalletAddress := common.HexToAddress(players[0].WalletAddress)
	player2WalletAddress := common.HexToAddress(players[1].WalletAddress)
	player1TemporaryAddress := common.HexToAddress(players[0].TemporaryAddress)
	player2TemporaryAddress := common.HexToAddress(players[1].TemporaryAddress)
	roundTimeoutBigInt := big.NewInt(roundTimeout)
	maxRoundsBigInt := big.NewInt(maxRounds)
	initialHPBigInt := big.NewInt(initialHP)
	retryCnt := 3
	for {
		select {
		case <-c.ctx.Done():
			return errors.New("create room contract failed, context canceled")
		default:
			if retryCnt == 0 {
				return errors.New("send create room tx failed")
			}
			retryCnt--
			txHash, err := c.roomMgrClient.sendCreateRoomTx(player1WalletAddress, player2WalletAddress, player1TemporaryAddress, player2TemporaryAddress,
				roundTimeoutBigInt, maxRoundsBigInt, initialHPBigInt)
			if err != nil {
				log.Errorw("send create room tx failed", "err", err)
				// not retriable error
				if strings.Contains(strings.ToLower(err.Error()), "revert") {
					return err
				}
				continue
			}
			log.Infow("createRoomContract: create room contract success", "tx hash", txHash, "game id", gameID)
			c.createRoomTxToGameID.Set(txHash, fmt.Sprint(gameID), int(time.Hour.Seconds()))
			createRoomTxModel := &dao.CreateRoomTx{
				GameID:       gameID,
				Status:       dao.TxStatusSent,
				TxHash:       txHash,
				RoundTimeout: time.Duration(roundTimeout) * time.Second,
				MaxRounds:    uint64(maxRounds),
			}
			err = db.SaveCreateRoomTx(createRoomTxModel)
			if err != nil {
				log.Errorw("save create room tx failed", "err", err)
				continue
			}
		}
	}

}

func (c *Chain) setRoundReady(gameID uint, roundNumber uint32, roomContractHex string) error {
	roomContractAddress := common.HexToAddress(roomContractHex)
	retryCnt := 3
	for {
		select {
		case <-c.ctx.Done():
			return errors.New("create room contract failed, context canceled")
		default:
			if retryCnt == 0 {
				return errors.New("send create room tx failed")
			}
			retryCnt--
			txHash, err := c.roomMgrClient.sendStartANewRound(roomContractAddress)
			if err != nil {
				log.Errorw("send set round read tx failed", "err", err)
				// not retriable error
				if strings.Contains(strings.ToLower(err.Error()), "revert") {
					return err
				}
				continue
			}
			setRoundReadyTxModel := &dao.SetRoundReadyTx{
				GameID:          gameID,
				Status:          dao.TxStatusSent,
				ContractAddress: roomContractHex,
				RoundNumber:     uint64(roundNumber),
				TxHash:          txHash,
			}
			err = db.SaveSetRoundReadyTx(setRoundReadyTxModel)
			if err != nil {
				log.Errorw("save set round ready tx failed", "err", err)
				continue
			}
		}
	}
}

func (c *Chain) handleChainEvents(evt *batchTxEvent) {
	batchTx := &types.EventBatch{}
	blockHash := strings.ToLower(hexutil.Encode(evt.blockHash))
	blockNumber := evt.blockNum
	blockTime := int64(evt.txs.Timestamp)
	for _, protoTx := range evt.txs.Transactions {
		hash := strings.ToLower(hexutil.Encode(protoTx.TxHash))
		switch tx := protoTx.Tx.(type) {
		case *proto.Transaction_RoomContractCreated:
			// maybe stale event, just skip
			gidStr, err := c.createRoomTxToGameID.Get(hash)
			if err != nil {
				log.Errorf("createRoomTxToGameID: load tx with hash %s from cache failed: %s", hash, err.Error())
				continue
			}
			gid, err := strconv.Atoi(gidStr)
			if err != nil {
				log.Errorf("createRoomTxToGameID: decoded loaded tx with hash %s failed: %s", hash, err.Error())
				continue
			}
			log.Infof("contractCreated: gameID %d, blockHash %s, blockNumber %d, tx %s", gid, blockHash, blockNumber, hash)
			c.contractCreated(batchTx, blockTime, uint(gid), blockHash, blockNumber, tx)
		case *proto.Transaction_RoomContractSetupReady:
			gid, err := c.getRoomIDByContract(tx.RoomContractSetupReady.RoomContractAddress)
			if err != nil {
				log.Errorf("cannot find room contract tx with contract hash %s, err: %s", err.Error())
				continue
			}
			log.Infof("contractSetupReady: gameID %d, blockHash %s, blockNumber %d, tx %s", gid, blockHash, blockNumber, hash)
			c.roundSetupCompleted(batchTx, blockTime, gid, blockHash, blockNumber, tx)
		case *proto.Transaction_CommitmentsOnChain:
			gid, err := c.getRoomIDByContract(tx.CommitmentsOnChain.RoomContractAddress)
			if err != nil {
				log.Errorf("cannot find room contract tx with contract hash %s, err: %s", err.Error())
				continue
			}
			address := types.PlayerAddress{}
			address.FromProto(tx.CommitmentsOnChain.Address)
			log.Infof("commitmentOnChain: gameID %d, blockHash %s, blockNumber %d, tx %s, address %s", gid, blockHash, blockNumber, hash, address.String())
			c.commitmentOnChain(batchTx, blockTime, gid, hash, blockHash, blockNumber, tx)
		case *proto.Transaction_CardsOnChain:
			gid, err := c.getRoomIDByContract(tx.CardsOnChain.RoomContractAddress)
			if err != nil {
				log.Errorf("cannot find room contract tx with contract hash %s, err: %s", err.Error())
				continue
			}
			address := types.PlayerAddress{}
			address.FromProto(tx.CardsOnChain.Address)
			log.Infof("cardsOnChain: gameID %d, blockHash %s, blockNumber %d, tx %s, contract address: %s, player address %s", gid, blockHash, blockNumber, hash, tx.CardsOnChain.RoomContractAddress, address.String())
			c.cardsOnChain(batchTx, blockTime, gid, hash, blockHash, blockNumber, tx)
		}
	}
	batchTx.Wait()
}

func (c *Chain) getRoomIDByContract(contractAddress string) (uint, error) {
	contractAddress = strings.ToLower(contractAddress)
	gidStr, err := c.gameContractToRoomID.Get(contractAddress)
	if err == nil {
		gid, err := strconv.Atoi(gidStr)
		if err == nil {
			return uint(gid), nil
		}
	}
	dbRoom, err := db.GetCreateRoomTxByContract(contractAddress)
	if err != nil {
		return 0, err
	}
	c.gameContractToRoomID.Set(contractAddress, fmt.Sprint(dbRoom.GameID), int(time.Hour.Seconds()))
	return dbRoom.GameID, nil
}

func (c *Chain) contractCreated(batchTx *types.EventBatch, blockTime int64, gameID uint, blockHash string, blockNumber uint64, tx *proto.Transaction_RoomContractCreated) error {
	roomContract := strings.ToLower(tx.RoomContractCreated.RoomContractAddress)
	contractCreatedEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &types.RoomContractCreated{
		GameID:              gameID,
		RoomContractAddress: roomContract,
		TimeStamp:           blockTime,
	}, true)
	batchTx.Add(contractCreatedEvt)
	c.workerManager.SendEvent(fmt.Sprint(gameID), contractCreatedEvt)
	c.gameContractToRoomID.Set(roomContract, fmt.Sprint(gameID), int(time.Hour.Seconds()))
	err := c.createRoomTxToGameID.Delete(roomContract)
	if err != nil {
		log.Errorf("createRoomTxToGameID: delete tx with hash %s from cache failed: %s", roomContract, err.Error())
	}
	return db.UpdateCreateRoomTxBlockHashAndContractByGameID(gameID, blockHash, blockNumber, roomContract)
}

func (c *Chain) roundSetupCompleted(batchTx *types.EventBatch, blockTime int64, gameID uint, blockHash string, blockNumber uint64, tx *proto.Transaction_RoomContractSetupReady) error {
	roundNumber := tx.RoomContractSetupReady.RoundNumber
	roundSetupCompletedEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.NewRoundSetupComplete{
		GameID:      gameID,
		RoundNumber: roundNumber,
		TimeStamp:   blockTime,
	}, true)
	batchTx.Add(roundSetupCompletedEvent)
	c.workerManager.SendEvent(fmt.Sprint(gameID), roundSetupCompletedEvent)
	return db.UpdateSetRoundReadyTxBlockHashByGameID(gameID, blockHash, blockNumber, roundNumber)
}

func (c *Chain) commitmentOnChain(batchTx *types.EventBatch, blockTime int64, gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_CommitmentsOnChain) error {
	player := types.PlayerAddress{}
	player.FromProto(tx.CommitmentsOnChain.Address)
	roundNumber := tx.CommitmentsOnChain.RoundNumber
	commitment := tx.CommitmentsOnChain.Commitment
	commitmentOnChainEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCommitmentOnChain{
		GameID:      gameID,
		Address:     player,
		RoundNumber: roundNumber,
		Commitment:  commitment,
		TimeStamp:   blockTime,
	}, true)
	batchTx.Add(commitmentOnChainEvent)
	c.workerManager.SendEvent(fmt.Sprint(gameID), commitmentOnChainEvent)
	return db.SaveCommitmentOnChainTx(&dao.CommitmentOnChainTx{
		GameID:           gameID,
		ContractAddress:  tx.CommitmentsOnChain.RoomContractAddress,
		TxHash:           txHash,
		BlockHash:        blockHash,
		BlockNumber:      blockNumber,
		Status:           dao.TxStatusSent,
		RoundNumber:      uint64(roundNumber),
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	})
}

func (c *Chain) cardsOnChain(batchTx *types.EventBatch, blockTime int64, gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_CardsOnChain) error {
	player := types.PlayerAddress{}
	player.FromProto(tx.CardsOnChain.Address)
	roundNumber := tx.CardsOnChain.RoundNumber
	salt := tx.CardsOnChain.Salt
	cardsUint := make([]uint, len(tx.CardsOnChain.Cards))
	for i := range cardsUint {
		cardsUint[i] = uint(tx.CardsOnChain.Cards[i])
	}
	cardsOnChainEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCardsOnChain{
		GameID:      gameID,
		Address:     player,
		RoundNumber: roundNumber,
		Salt:        salt,
		Cards:       cardsUint,
		TimeStamp:   blockTime,
	}, true)
	batchTx.Add(cardsOnChainEvent)
	c.workerManager.SendEvent(fmt.Sprint(gameID), cardsOnChainEvent)
	return db.SaveCardsOnChainTx(&dao.CardsOnChainTx{
		GameID:           gameID,
		ContractAddress:  tx.CardsOnChain.RoomContractAddress,
		TxHash:           txHash,
		BlockHash:        blockHash,
		BlockNumber:      blockNumber,
		Status:           dao.TxStatusSent,
		RoundNumber:      uint64(roundNumber),
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	})
}
