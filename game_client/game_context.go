package gameclient

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type playerState uint8

const (
	playerStateIdle playerState = iota
	playerStateWattingGameMatched
	playerStateWaittingConfirm
	playerStateWaittingGameReady
	playerStateWaittingCommitmentsOnChain
	playerStateWaittingCommitmentsSubmitted
	playerStateWaittingCardsSubmitted
	playerStateWaittingCardsOnChain
	playerStateWaittingRoundEnd
)

func (ps playerState) String() string {
	switch ps {
	case playerStateIdle:
		return "idle"
	case playerStateWattingGameMatched:
		return "waiting game matched"
	case playerStateWaittingConfirm:
		return "waiting confirm"
	case playerStateWaittingGameReady:
		return "waiting game ready"
	case playerStateWaittingCommitmentsOnChain:
		return "waiting commitments on chain"
	case playerStateWaittingCommitmentsSubmitted:
		return "waiting commitments submitted"
	case playerStateWaittingCardsSubmitted:
		return "waiting cards submitted"
	case playerStateWaittingCardsOnChain:
		return "waiting cards on chain"
	case playerStateWaittingRoundEnd:
		return "waiting round end"
	}
	return "unknown state"
}

type GameContext struct {
	ctx             context.Context
	wallet          *wallet.Wallet
	temporaryWallet *wallet.Wallet
	chainClient     *ethclient.Client
	rpcClient       GameClient

	gameID                uint
	players               []*types.PlayerAddress
	currentRound          uint32
	contractAddress       string
	myself                *types.PlayerAddress
	contract              *contract.RoomContract
	bindOpts              *bind.TransactOpts
	evtChan               chan *proto.Event
	errChan               chan error
	state                 playerState
	lock                  sync.Mutex
	commitmentOnChainChan chan struct{}
	roomReadyChan         chan struct{}
	cardOnChainChan       chan struct{}
}

type GameContextConfig struct {
	Wallet          *wallet.Wallet
	TemporaryWallet *wallet.Wallet
	ChainClient     *ethclient.Client
	RpcClient       *rpc.Client
	HttpClient      *HttpClient
}

func NewGameContext(ctx context.Context, cfg *GameContextConfig) (*GameContext, error) {
	chainId, err := cfg.ChainClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	cfg.TemporaryWallet.BuildTxSinger(chainId)
	bindOpts := &bind.TransactOpts{
		From:   cfg.TemporaryWallet.GetAddr(),
		Signer: cfg.TemporaryWallet.BuildTxSinger(chainId),
	}

	var gameClient GameClient
	if cfg.RpcClient != nil {
		gameClient = WrapRpcClient(cfg.RpcClient)
	} else if cfg.HttpClient != nil {
		gameClient = WrapHttpClient(cfg.HttpClient)
	}
	return &GameContext{
		ctx:                   ctx,
		myself:                types.NewPlayerAddress(cfg.Wallet.GetAddrHex(), cfg.TemporaryWallet.GetAddrHex()),
		wallet:                cfg.Wallet,
		temporaryWallet:       cfg.TemporaryWallet,
		evtChan:               make(chan *proto.Event, 10),
		errChan:               make(chan error, 10),
		bindOpts:              bindOpts,
		chainClient:           cfg.ChainClient,
		rpcClient:             gameClient,
		commitmentOnChainChan: make(chan struct{}),
		roomReadyChan:         make(chan struct{}),
		cardOnChainChan:       make(chan struct{}),
	}, nil
}

func (c *GameContext) Run() error {
	err := c.rpcClient.Subscribe(c.myself.String(), c.myself.String(), c.evtChan, c.errChan)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case err := <-c.errChan:
				fmt.Println("subscribe err: ", err)
			case evt, ok := <-c.evtChan:
				if !ok {
					return
				}
				switch evt.Type {
				case proto.EventType_TYPE_KNOWN:
					return
				case proto.EventType_TYPE_MATCHED:
					fmt.Println("game matched, please confirm")
					phase, err := c.rpcClient.GetGamePhase(c.ctx, c.myself)
					if err != nil {
						fmt.Println("error: ", err.Error())
						continue
					}
					c.lock.Lock()
					c.gameID = phase.GameID()
					c.players = phase.Players()
					c.state = playerStateWaittingConfirm
					c.currentRound = 1
					c.lock.Unlock()
				case proto.EventType_TYPE_PART_CONFIRMED:
					fmt.Println("player part confirmed")
				case proto.EventType_TYPE_GAME_CREATED:
					fmt.Println("game created, please submit cards")
					// get contract
					phase, err := c.rpcClient.GetGamePhase(c.ctx, c.myself)
					if err != nil {
						fmt.Println("error: ", err.Error())
					}
					c.lock.Lock()
					c.gameID = phase.GameID()
					c.contractAddress = phase.ContractAddress()
					c.contract, err = contract.NewRoomContract(common.HexToAddress(c.contractAddress), c.chainClient)
					if err != nil {
						fmt.Println("error: ", err.Error())
					}
					c.currentRound = 1
					fmt.Println("contract address: ", c.contractAddress)
					c.state = playerStateWaittingCommitmentsSubmitted
					c.lock.Unlock()
				case proto.EventType_TYPE_ROUND_READY:
					fmt.Println("round ready, please submit cards")
					c.lock.Lock()
					c.state = playerStateWaittingCommitmentsSubmitted
					c.lock.Unlock()
				case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
					fmt.Println("commitments on chain")
					c.lock.Lock()
					c.state = playerStateWaittingCardsSubmitted
					c.lock.Unlock()
					c.commitmentOnChainChan <- struct{}{}
				case proto.EventType_TYPE_CARDS_ON_CHAIN:
					c.lock.Lock()
					fmt.Println("cards on chain")
					c.state = playerStateWaittingRoundEnd
					c.lock.Unlock()
				case proto.EventType_TYPE_ROUND_COMPLETE:
					fmt.Println("round complete, please confirm to start a new round")
					battleInfo, err := c.rpcClient.GetBattleInfo(c.ctx, c.gameID, uint(c.currentRound))
					if err != nil {
						fmt.Println("error: ", err.Error())
					}
					fmt.Println("round result: ", types.ToJsonLoggable(battleInfo.RoundResult()))
					if !battleInfo.IsGameOver() {
						c.lock.Lock()
						c.currentRound++
						c.state = playerStateWaittingConfirm
						c.lock.Unlock()
					} else {
						fmt.Println("game result: ", types.ToJsonLoggable(battleInfo.GameResult()))
					}
				case proto.EventType_TYPE_GAME_COMPLETE:
					fmt.Println("game complete")
					c.lock.Lock()
					c.state = playerStateIdle
					c.lock.Unlock()
				}
			}
		}
	}()
	return nil
}

func (c *GameContext) JoinQueue() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.state != playerStateIdle {
		return fmt.Errorf("cannot join queue, invalid state: %s", c.state.String())
	}

	err := c.rpcClient.JoinQueue(c.ctx, c.myself)
	if err != nil {
		return err
	}
	c.state = playerStateWattingGameMatched
	return nil
}

func (c *GameContext) ExitQueue() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.state != playerStateWattingGameMatched {
		return fmt.Errorf("cannot exit queue, invalid state: %s", c.state.String())
	}
	err := c.rpcClient.ExitQueue(c.ctx, c.myself)
	if err != nil {
		return err
	}
	c.state = playerStateIdle
	return nil
}

func (c *GameContext) ConfirmBattle() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.state != playerStateWaittingConfirm {
		return fmt.Errorf("cannot confirm battle, invalid state: %s", c.state.String())
	}
	err := c.rpcClient.ConfirmBattle(c.ctx, c.myself, c.gameID, uint(c.currentRound))
	if err != nil {
		return err
	}
	c.state = playerStateWaittingGameReady
	return nil
}

func (c *GameContext) SubmitCommitment(cardHash []byte) (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.state != playerStateWaittingCommitmentsSubmitted {
		return "", fmt.Errorf("cannot submit commitment, invalid state: %s", c.state.String())
	}
	fmt.Println("submit commitment, round: ", c.currentRound)
	tx, err := c.contract.SubmitCardsHash(c.bindOpts, [32]byte(cardHash), big.NewInt(int64(c.currentRound)))
	if err != nil {
		return "", err
	}
	c.state = playerStateWaittingCommitmentsOnChain
	return tx.Hash().String(), nil
}

func (c *GameContext) SubmitCards(cards string, salt string) (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	// if c.state != playerStateWaittingCardsSubmitted {
	// 	return "", fmt.Errorf("cannot submit cards, invalid state: %s", c.state.String())
	// }
	fmt.Println("submit cards, round: ", c.currentRound)
	tx, err := c.contract.SubmitCards(c.bindOpts, cards, salt, big.NewInt(int64(c.currentRound)))
	if err != nil {
		return "", err
	}
	c.state = playerStateWaittingCardsOnChain
	return tx.Hash().String(), nil
}

func (c *GameContext) Continue() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.state != playerStateIdle {
		return fmt.Errorf("cannot continue, invalid state: %s", c.state.String())
	}
	err := c.rpcClient.ContinueGame(c.ctx, c.myself, c.gameID)
	if err != nil {
		return err
	}
	c.state = playerStateWaittingGameReady
	return nil
}

func (c *GameContext) WaitCommitmentOnChain() {
	<-c.commitmentOnChainChan
}
