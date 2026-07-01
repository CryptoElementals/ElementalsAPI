package worker

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const roomV3SingleTxGasLimit = 500_000

// concurrentRoomV3Client is a helper around the RoomV3 contract that mirrors the
// behaviour of the JavaScript example in js_example.js by packing individual
// actions into encoded "tasks" and submitting them via batchSubmitTasks.
type concurrentRoomV3Client struct {
	client    *ethclient.Client
	roomV3Ctr *contract.RoomV3Contract
	optsPool  chan *bind.TransactOpts
	wallets   []*wallet.Wallet
}

// newConcurrentRoomV3Client constructs a RoomV3 client that can safely submit
// transactions concurrently using a pool of bind options (one per wallet).
func newConcurrentRoomV3Client(
	ctx context.Context,
	client *ethclient.Client,
	roomV3Address string,
	wallets []*wallet.Wallet,
	chainID int64,
	isDevelop ...bool,
) (*concurrentRoomV3Client, error) {
	roomV3Ctr, err := contract.NewRoomV3Contract(common.HexToAddress(roomV3Address), client)
	if err != nil {
		return nil, fmt.Errorf("newRoomV3Contract: create room v3 contract failed: %s", err.Error())
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
			GasLimit: roomV3SingleTxGasLimit,
			Nonce:    new(big.Int).SetUint64(nonce),
		}
		if len(isDevelop) != 0 && isDevelop[0] {
			bindOpts.NoSend = true
		}
		optsPool <- bindOpts
	}

	return &concurrentRoomV3Client{
		client:    client,
		roomV3Ctr: roomV3Ctr,
		wallets:   wallets,
		optsPool:  optsPool,
	}, nil
}

// submitTasks sends the given tasks slice using RoomV3.batchSubmitTasks,
// handling nonce management and gas limit scaling. All task encoding is expected
// to be done by the caller.
func (c *concurrentRoomV3Client) submitTasks(indexes []uint8, tasks [][]byte) (string, error) {
	if len(tasks) == 0 {
		return "", fmt.Errorf("no tasks to submit")
	}

	bindOpts := <-c.optsPool
	var txSent bool
	defer func() {
		if txSent && !bindOpts.NoSend && bindOpts.Nonce != nil {
			bindOpts.Nonce = new(big.Int).Add(bindOpts.Nonce, big.NewInt(1))
		}
		c.optsPool <- bindOpts
	}()

	// Scale gas limit with number of tasks.
	bindOpts.GasLimit = uint64(len(tasks) * roomV3SingleTxGasLimit)

	tx, err := c.roomV3Ctr.BatchSubmitTasks(bindOpts, indexes, tasks)
	if err != nil {
		log.Errorf("sendBatchTasks: batchSubmitTasks failed: %s", err.Error())
		return "", fmt.Errorf("batchSubmitTasks failed: %s", err.Error())
	}
	txSent = true

	txHash := strings.ToLower(tx.Hash().String())
	log.Debugw("sendBatchTasks: batch submitted",
		"task_count", len(tasks),
		"tx_hash", txHash)
	return txHash, nil
}

// --- ABI encoding helpers ---

func mustNewType(t string) abi.Type {
	typ, err := abi.NewType(t, "", nil)
	if err != nil {
		panic(fmt.Sprintf("abi.NewType(%q) failed: %v", t, err))
	}
	return typ
}

func EncodeCreateRoomTask(
	player1ID, player2ID *big.Int,
	player1TemporaryAddress, player2TemporaryAddress common.Address,
	roundTimeout, totalRound, totalCardIndex, initialHP, gameID, tournamentID, tierNo *big.Int,
) ([]byte, error) {
	args := abi.Arguments{
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("address")},
		{Type: mustNewType("address")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
	}
	return args.Pack(
		player1ID,
		player2ID,
		player1TemporaryAddress,
		player2TemporaryAddress,
		roundTimeout,
		totalRound,
		totalCardIndex,
		initialHP,
		gameID,
		tournamentID,
		tierNo,
	)
}

func EncodeSubmitCardHashTask(
	gameID *big.Int,
	cardHash [32]byte,
	cardIndex *big.Int,
	round *big.Int,
	signature []byte,
) ([]byte, error) {
	args := abi.Arguments{
		{Type: mustNewType("uint256")},
		{Type: mustNewType("bytes32")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("bytes")},
	}
	return args.Pack(
		gameID,
		cardHash,
		cardIndex,
		round,
		signature,
	)
}

func EncodeSubmitCardTask(
	gameID *big.Int,
	card *big.Int,
	salt []byte,
	cardIndex *big.Int,
	round *big.Int,
	signature []byte,
) ([]byte, error) {
	salt32 := leftPadToBytes32(salt)
	args := abi.Arguments{
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("bytes32")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("uint256")},
		{Type: mustNewType("bytes")},
	}
	return args.Pack(
		gameID,
		card,
		salt32,
		cardIndex,
		round,
		signature,
	)
}

func EncodeStartNewTurnTask(
	gameID *big.Int,
) ([]byte, error) {
	args := abi.Arguments{
		{Type: mustNewType("uint256")},
	}
	return args.Pack(gameID)
}

// leftPadToBytes32 mirrors ethers.zeroPadValue(toUtf8Bytes(x), 32) by left-padding
// the provided bytes with zeros up to 32 bytes (or truncating from the left if
// longer than 32 bytes).
func leftPadToBytes32(b []byte) [32]byte {
	var out [32]byte
	if len(b) >= 32 {
		copy(out[:], b[len(b)-32:])
		return out
	}
	offset := 32 - len(b)
	copy(out[offset:], b)
	return out
}
