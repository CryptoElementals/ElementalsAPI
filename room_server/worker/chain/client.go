package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
)

type concurrentRoomClient struct {
	client   bind.ContractBackend
	roomCtr  *contract.RoomManagerContract
	optsPool chan *bind.TransactOpts
	wallets  []*wallet.Wallet
}

func newConcurrentRoomClient(
	ctx context.Context,
	client bind.ContractBackend,
	roomMgrAddress string,
	wallets []*wallet.Wallet,
	chainID int64,
	isDevelop ...bool,
) (*concurrentRoomClient, error) {
	roomCtr, err := contract.NewRoomManagerContract(common.HexToAddress(roomMgrAddress), client)
	if err != nil {
		return nil, fmt.Errorf("newRoomManagerContract: create room contract failed: %s", err.Error())
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
	return &concurrentRoomClient{
		client:   client,
		roomCtr:  roomCtr,
		wallets:  wallets,
		optsPool: optsPool,
	}, nil
}

func (c *concurrentRoomClient) sendCreateRoomTx(
	player1WalletAddress, player2WalletAddress, player1TemporaryAddress, player2TemporaryAddress common.Address,
	roundTimeoutBigInt, maxRoundsBigInt, initialHPBigInt *big.Int,
) (string, error) {
	bindOpts := <-c.optsPool
	defer func() {
		c.optsPool <- bindOpts
	}()
	tx, err := c.roomCtr.CreateRoom(bindOpts, player1WalletAddress, player2WalletAddress,
		player1TemporaryAddress, player2TemporaryAddress, roundTimeoutBigInt, maxRoundsBigInt, initialHPBigInt)
	if err != nil {
		log.Errorf("createRoomContract: create room contract failed: %s", err.Error())
		return "", fmt.Errorf("create room contract failed: %s", err.Error())
	}
	return strings.ToLower(tx.Hash().String()), nil
}

func (c *concurrentRoomClient) sendStartANewRound(roomContractAddress common.Address) (string, error) {
	bindOpts := <-c.optsPool
	defer func() {
		c.optsPool <- bindOpts
	}()
	roomContract, err := contract.NewRoomContract(roomContractAddress, c.client)
	if err != nil {
		return "", fmt.Errorf("newRoomContract: create room contract failed: %s", err.Error())
	}
	tx, err := roomContract.StartANewRound(bindOpts)
	if err != nil {
		return "", err
	}
	txHash := strings.ToLower(tx.Hash().String())
	return txHash, nil
}
