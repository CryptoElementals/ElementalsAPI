package chain

import (
	"context"
	"encoding/hex"
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
)

type concurrentRoomV2Client struct {
	client    bind.ContractBackend
	roomV2Ctr *contract.RoomV2Contract
	optsPool  chan *bind.TransactOpts
	wallets   []*wallet.Wallet
}

func newConcurrentRoomV2Client(
	ctx context.Context,
	client bind.ContractBackend,
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
		bindOpts := &bind.TransactOpts{
			Context: ctx,
			From:    w.GetAddr(),
			Signer:  w.BuildTxSinger(big.NewInt(chainID)),
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
	player1Bytes8, player2Bytes8 [8]byte,
	player1TemporaryAddress, player2TemporaryAddress common.Address,
	roundTimeout, totalRound, totalCardIndex, initialHP, gameID *big.Int,
) (string, error) {
	bindOpts := <-c.optsPool
	defer func() {
		c.optsPool <- bindOpts
	}()

	tx, err := c.roomV2Ctr.CreateRoom(
		bindOpts,
		player1Bytes8,
		player2Bytes8,
		player1TemporaryAddress,
		player2TemporaryAddress,
		roundTimeout,
		totalRound,
		totalCardIndex,
		initialHP,
		gameID,
	)
	if err != nil {
		log.Errorf("sendCreateRoomTx: create room failed: %s", err.Error())
		return "", fmt.Errorf("create room failed: %s", err.Error())
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

	// Convert player IDs (int64) to bytes8 for contract call
	var player1Bytes8 [8]byte
	var player2Bytes8 [8]byte
	// Convert int64 ID to [8]byte (big-endian)
	player1IDBig := big.NewInt(evt.Players[0].Id)
	player2IDBig := big.NewInt(evt.Players[1].Id)
	player1Bytes := player1IDBig.Bytes()
	player2Bytes := player2IDBig.Bytes()
	// Copy to bytes8 arrays (right-aligned, big-endian)
	copy(player1Bytes8[8-len(player1Bytes):], player1Bytes)
	copy(player2Bytes8[8-len(player2Bytes):], player2Bytes)

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
				player1Bytes8,
				player2Bytes8,
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
			log.Infow("createNewRoom: create new room success", "tx hash", txHash, "game id", evt.GameID)
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
	defer func() {
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
	}

	tx, err := c.roomV2Ctr.BatchSubmitCardsHash(bindOpts, gameIDs, commitments, cardIndexes, rounds, signatures)
	if err != nil {
		log.Errorf("sendBatchSubmitCardsHash: batch submit cards hash failed: %s", err.Error())
		return "", fmt.Errorf("batch submit cards hash failed: %s", err.Error())
	}
	return strings.ToLower(tx.Hash().String()), nil
}

// sendBatchSubmitCards submits multiple cards as a batch
func (c *concurrentRoomV2Client) sendBatchSubmitCards(events []*types.SubmitPlayerCard) (string, error) {
	if len(events) == 0 {
		return "", fmt.Errorf("no events to submit")
	}

	bindOpts := <-c.optsPool
	defer func() {
		c.optsPool <- bindOpts
	}()

	// Prepare batch arrays
	gameIDs := make([]*big.Int, len(events))
	cards := make([]string, len(events))
	salts := make([]string, len(events))
	cardIndexes := make([]*big.Int, len(events))
	rounds := make([]*big.Int, len(events))
	signatures := make([][]byte, len(events))

	for i, evt := range events {
		// Convert card to string
		cardStr := fmt.Sprintf("%d", evt.Card)
		// Convert salt bytes to hex string for proper encoding
		saltStr := hex.EncodeToString(evt.Salt)

		gameIDs[i] = big.NewInt(int64(evt.GameID))
		cards[i] = cardStr
		salts[i] = saltStr
		cardIndexes[i] = big.NewInt(int64(evt.CardIndex))
		rounds[i] = big.NewInt(int64(evt.RoundNumber))
		signatures[i] = evt.Signature
	}

	tx, err := c.roomV2Ctr.BatchSubmitCards(bindOpts, gameIDs, cards, salts, cardIndexes, rounds, signatures)
	if err != nil {
		log.Errorf("sendBatchSubmitCards: batch submit cards failed: %s", err.Error())
		return "", fmt.Errorf("batch submit cards failed: %s", err.Error())
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

	return c.retryBatchSubmission(
		func() (string, error) {
			return c.roomV2Client.sendBatchSubmitCardsHash(events)
		},
		"submit player commitments batch",
		len(events),
	)
}

// submitPlayerCardsBatch submits multiple player cards in a batch
func (c *Chain) submitPlayerCardsBatch(events []*types.SubmitPlayerCard) error {
	if c.roomV2Client == nil {
		return errors.New("room v2 client not initialized")
	}

	if len(events) == 0 {
		return nil
	}

	return c.retryBatchSubmission(
		func() (string, error) {
			return c.roomV2Client.sendBatchSubmitCards(events)
		},
		"submit player cards batch",
		len(events),
	)
}

// sendStartANewCard sends StartANewCard transaction to RoomV2 contract (this is actually startANewTurn)
func (c *concurrentRoomV2Client) sendStartANewCard(gameID uint) (string, error) {
	bindOpts := <-c.optsPool
	defer func() {
		c.optsPool <- bindOpts
	}()

	gameIDBigInt := big.NewInt(int64(gameID))
	tx, err := c.roomV2Ctr.StartANewCard(bindOpts, gameIDBigInt)
	if err != nil {
		log.Errorf("sendStartANewCard: start a new card failed: %s", err.Error())
		return "", fmt.Errorf("start a new card failed: %s", err.Error())
	}
	return strings.ToLower(tx.Hash().String()), nil
}

// startANewTurn starts a new turn (card) on the RoomV2 contract
func (c *Chain) startANewTurn(gameID uint) error {
	if c.roomV2Client == nil {
		return errors.New("room v2 client not initialized")
	}

	return c.retryBatchSubmission(
		func() (string, error) {
			return c.roomV2Client.sendStartANewCard(gameID)
		},
		"start a new turn",
		1,
	)
}

// retryBatchSubmission is a helper function that retries a batch submission operation
func (c *Chain) retryBatchSubmission(submitFn func() (string, error), operationName string, count int) error {
	retryCnt := 3
	for {
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("%s failed, context canceled", operationName)
		default:
			if retryCnt == 0 {
				return fmt.Errorf("%s failed after retries", operationName)
			}
			retryCnt--
			txHash, err := submitFn()
			if err != nil {
				log.Errorw(fmt.Sprintf("send %s tx failed", operationName), "err", err, "count", count)
				// not retriable error
				if strings.Contains(strings.ToLower(err.Error()), "revert") {
					return err
				}
				time.Sleep(time.Second)
				continue
			}
			log.Infow(fmt.Sprintf("%s: success", operationName), "tx hash", txHash, "count", count)
			return nil
		}
	}
}
