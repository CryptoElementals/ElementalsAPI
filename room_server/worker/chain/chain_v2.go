package chain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

const SingleTxGasLimit = 500_000

type concurrentRoomV2Client struct {
	client    *ethclient.Client
	roomV2Ctr *contract.RoomV2Contract
	optsPool  chan *bind.TransactOpts
	wallets   []*wallet.Wallet
}

func newConcurrentRoomV2Client(
	ctx context.Context,
	client *ethclient.Client,
	roomV2Address string,
	wallets []*wallet.Wallet,
	chainID int64,
	isDevelop ...bool,
) (*concurrentRoomV2Client, error) {
	roomV2Ctr, err := contract.NewRoomV2Contract(common.HexToAddress(roomV2Address), client)
	if err != nil {
		return nil, fmt.Errorf("newRoomV2Contract: create room v2 contract failed: %s", err.Error())
	}
	optsPool := make(chan *bind.TransactOpts, len(wallets))
	for _, w := range wallets {
		nonce, err := client.PendingNonceAt(ctx, w.GetAddr())
		if err != nil {
			return nil, fmt.Errorf("pending nonce for wallet %s: %w", w.GetAddr().Hex(), err)
		}
		bindOpts := &bind.TransactOpts{
			Context:  ctx,
			From:     w.GetAddr(),
			Signer:   w.BuildTxSinger(big.NewInt(chainID)),
			GasLimit: 500_000,
			Nonce:    new(big.Int).SetUint64(nonce),
		}
		if len(isDevelop) != 0 && isDevelop[0] {
			bindOpts.NoSend = true
		}
		optsPool <- bindOpts
	}
	return &concurrentRoomV2Client{
		client:    client,
		roomV2Ctr: roomV2Ctr,
		wallets:   wallets,
		optsPool:  optsPool,
	}, nil
}

// sendCreateRoomTx sends CreateRoom transaction to RoomV2 contract
func (c *concurrentRoomV2Client) sendCreateRoomTx(
	player1ID, player2ID *big.Int,
	player1TemporaryAddress, player2TemporaryAddress common.Address,
	roundTimeout, totalRound, totalCardIndex, initialHP, gameID *big.Int,
) (string, error) {
	bindOpts := <-c.optsPool
	var sendErr error
	defer func() {
		if sendErr == nil && !bindOpts.NoSend && bindOpts.Nonce != nil {
			bindOpts.Nonce = new(big.Int).Add(bindOpts.Nonce, big.NewInt(1))
		}
		c.optsPool <- bindOpts
	}()

	tx, sendErr := c.roomV2Ctr.CreateRoom(
		bindOpts,
		player1ID,
		player2ID,
		player1TemporaryAddress,
		player2TemporaryAddress,
		roundTimeout,
		totalRound,
		totalCardIndex,
		initialHP,
		gameID,
	)
	if sendErr != nil {
		log.Errorf("sendCreateRoomTx: create room failed: %s", sendErr.Error())
		return "", fmt.Errorf("create room failed: %s", sendErr.Error())
	}
	return strings.ToLower(tx.Hash().String()), nil
}

func (c *Chain) SetTurnReady(evt *types.RequireSetupNewTurnEvent) error {
	// Use RoomV2 contract StartANewCard (which is actually startANewTurn)
	return c.startANewTurn(evt.GameID)
}

func (c *Chain) createNewRoom(evt *types.RequireGameCreationEvent) error {
	if c.roomV2Client == nil {
		return errors.New("room v2 client not initialized")
	}
	player1IDBig := big.NewInt(evt.Players[0].Id)
	player2IDBig := big.NewInt(evt.Players[1].Id)
	player1TemporaryAddress := common.HexToAddress(evt.Players[0].TemporaryAddress)
	player2TemporaryAddress := common.HexToAddress(evt.Players[1].TemporaryAddress)

	roundTimeoutBigInt := big.NewInt(evt.RoundTimeout)
	totalRoundBigInt := big.NewInt(evt.MaxRoundNumber)
	totalCardIndexBigInt := big.NewInt(3) // 3 cards per round
	initialHPBigInt := big.NewInt(evt.InitialHP)
	gameIdBigInt := big.NewInt(int64(evt.GameID))

	retryCnt := 3
	for {
		select {
		case <-c.ctx.Done():
			return errors.New("create new room failed, context canceled")
		default:
			if retryCnt == 0 {
				return errors.New("send create new room tx failed")
			}
			retryCnt--
			txHash, err := c.roomV2Client.sendCreateRoomTx(
				player1IDBig,
				player2IDBig,
				player1TemporaryAddress,
				player2TemporaryAddress,
				roundTimeoutBigInt,
				totalRoundBigInt,
				totalCardIndexBigInt,
				initialHPBigInt,
				gameIdBigInt,
			)
			if err != nil {
				log.Errorw("send create new room tx failed", "err", err)
				// not retriable error
				if strings.Contains(strings.ToLower(err.Error()), "revert") {
					return err
				}
				time.Sleep(time.Second)
				continue
			}
			c.recordTxStart(txHash, "createNewRoom", evt.GameID)
			// Transaction tables removed - no longer saving to database
			return nil
		}
	}
}

// sendBatchSubmitCardsHash submits multiple commitments as a batch
func (c *concurrentRoomV2Client) sendBatchSubmitCardsHash(events []*types.SubmitPlayerCommitment) (string, error) {
	if len(events) == 0 {
		return "", fmt.Errorf("no events to submit")
	}

	bindOpts := <-c.optsPool
	var sendErr error
	defer func() {
		if sendErr == nil && !bindOpts.NoSend && bindOpts.Nonce != nil {
			bindOpts.Nonce = new(big.Int).Add(bindOpts.Nonce, big.NewInt(1))
		}
		c.optsPool <- bindOpts
	}()

	// Prepare batch arrays
	gameIDs := make([]*big.Int, len(events))
	commitments := make([][32]byte, len(events))
	cardIndexes := make([]*big.Int, len(events))
	rounds := make([]*big.Int, len(events))
	signatures := make([][]byte, len(events))

	for i, evt := range events {
		// Convert commitment to bytes32
		if len(evt.Commitment) != 32 {
			return "", fmt.Errorf("commitment must be 32 bytes, got %d at index %d", len(evt.Commitment), i)
		}
		var commitmentHash [32]byte
		copy(commitmentHash[:], evt.Commitment)

		gameIDs[i] = big.NewInt(int64(evt.GameID))
		commitments[i] = commitmentHash
		cardIndexes[i] = big.NewInt(int64(evt.CommitmentIndex))
		rounds[i] = big.NewInt(int64(evt.RoundNumber))
		signatures[i] = evt.Signature
		log.Debugw("sendBatchSubmitCardsHash: batch submit cards hash",
			"game id", evt.GameID,
			"commitment index", evt.CommitmentIndex,
			"round number", evt.RoundNumber,
			"commitment", hexutil.Encode(evt.Commitment),
			"signature", hexutil.Encode(evt.Signature))
	}
	// increase gas limit by event number
	bindOpts.GasLimit = uint64(len(events) * SingleTxGasLimit)
	tx, sendErr := c.roomV2Ctr.BatchSubmitCardHashes(bindOpts, gameIDs, commitments, cardIndexes, rounds, signatures)
	if sendErr != nil {
		log.Errorf("sendBatchSubmitCardsHash: batch submit cards hash failed: %s", sendErr.Error())
		return "", fmt.Errorf("batch submit cards hash failed: %s", sendErr.Error())
	}
	return strings.ToLower(tx.Hash().String()), nil
}

// sendBatchSubmitCards submits multiple cards as a batch
func (c *concurrentRoomV2Client) sendBatchSubmitCards(events []*types.SubmitPlayerCard) (string, error) {
	if len(events) == 0 {
		return "", fmt.Errorf("no events to submit")
	}

	bindOpts := <-c.optsPool
	var sendErr error
	defer func() {
		if sendErr == nil && !bindOpts.NoSend && bindOpts.Nonce != nil {
			bindOpts.Nonce = new(big.Int).Add(bindOpts.Nonce, big.NewInt(1))
		}
		c.optsPool <- bindOpts
	}()

	// Prepare batch arrays
	gameIDs := make([]*big.Int, len(events))
	cards := make([]*big.Int, len(events))
	salts := make([]string, len(events))
	cardIndexes := make([]*big.Int, len(events))
	rounds := make([]*big.Int, len(events))
	signatures := make([][]byte, len(events))

	for i, evt := range events {
		// Convert card to string
		cardBigInt := big.NewInt(int64(evt.Card))
		// Convert salt bytes to hex string for proper encoding
		saltStr := string(evt.Salt)

		gameIDs[i] = big.NewInt(int64(evt.GameID))
		cards[i] = cardBigInt
		salts[i] = saltStr
		cardIndexes[i] = big.NewInt(int64(evt.CardIndex))
		rounds[i] = big.NewInt(int64(evt.RoundNumber))
		signatures[i] = evt.Signature
		log.Debugw("sendBatchSubmitCards: batch submit cards",
			"game id", evt.GameID,
			"card index", evt.CardIndex,
			"round number", evt.RoundNumber,
			"card", evt.Card,
			"signature", hexutil.Encode(evt.Signature))
	}
	// increase gas limit by event number
	bindOpts.GasLimit = uint64(len(events) * SingleTxGasLimit)
	tx, sendErr := c.roomV2Ctr.BatchSubmitCards(bindOpts, gameIDs, cards, salts, cardIndexes, rounds, signatures)
	if sendErr != nil {
		log.Errorf("sendBatchSubmitCards: batch submit cards failed: %s", sendErr.Error())
		return "", fmt.Errorf("batch submit cards failed: %s", sendErr.Error())
	}
	return strings.ToLower(tx.Hash().String()), nil
}

// submitPlayerCommitmentsBatch submits multiple player commitments in a batch
func (c *Chain) submitPlayerCommitmentsBatch(events []*types.SubmitPlayerCommitment) error {
	if c.roomV2Client == nil {
		return errors.New("room v2 client not initialized")
	}

	if len(events) == 0 {
		return nil
	}
	txHash, err := c.roomV2Client.sendBatchSubmitCardsHash(events)
	if err != nil {
		return err
	}
	c.recordTxStart(txHash, "submitPlayerCommitmentsBatch", 0)
	return nil
}

// submitPlayerCardsBatch submits multiple player cards in a batch
func (c *Chain) submitPlayerCardsBatch(events []*types.SubmitPlayerCard) error {
	if c.roomV2Client == nil {
		return errors.New("room v2 client not initialized")
	}

	if len(events) == 0 {
		return nil
	}
	txHash, err := c.roomV2Client.sendBatchSubmitCards(events)
	if err != nil {
		return err
	}
	c.recordTxStart(txHash, "submitPlayerCardsBatch", 0)
	return nil
}

// sendStartANewCard sends StartANewCard transaction to RoomV2 contract (this is actually startANewTurn)
func (c *concurrentRoomV2Client) sendStartANewCard(gameID uint) (string, error) {
	bindOpts := <-c.optsPool
	var sendErr error
	defer func() {
		if sendErr == nil && !bindOpts.NoSend && bindOpts.Nonce != nil {
			bindOpts.Nonce = new(big.Int).Add(bindOpts.Nonce, big.NewInt(1))
		}
		c.optsPool <- bindOpts
	}()

	gameIDBigInt := big.NewInt(int64(gameID))
	tx, sendErr := c.roomV2Ctr.StartANewTurn(bindOpts, gameIDBigInt)
	if sendErr != nil {
		log.Errorf("sendStartANewCard: start a new card failed: %s", sendErr.Error())
		return "", fmt.Errorf("start a new card failed: %s", sendErr.Error())
	}
	return strings.ToLower(tx.Hash().String()), nil
}

// startANewTurn starts a new turn (card) on the RoomV2 contract
func (c *Chain) startANewTurn(gameID uint) error {
	if c.roomV2Client == nil {
		return errors.New("room v2 client not initialized")
	}
	txHash, err := c.roomV2Client.sendStartANewCard(gameID)
	if err != nil {
		return err
	}
	c.recordTxStart(txHash, "startANewTurn", gameID)
	return nil
}
