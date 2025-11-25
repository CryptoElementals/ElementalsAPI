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
	temporaryWallet *wallet.Wallet
	chainClient     *ethclient.Client
	rpcClient       *rpc.Client

	gameID                uint
	players               []*types.PlayerAddress
	currentRound          uint32
	currentTurn           uint32
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

func NewGameContext(ctx context.Context, playerId int64, temporaryWallet *wallet.Wallet, chainClient *ethclient.Client, rpcClient *rpc.Client) (*GameContext, error) {
	chainId, err := chainClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	temporaryWallet.BuildTxSinger(chainId)
	bindOpts := &bind.TransactOpts{
		From:   temporaryWallet.GetAddr(),
		Signer: temporaryWallet.BuildTxSinger(chainId),
	}
	return &GameContext{
		ctx:                   ctx,
		myself:                types.NewPlayerAddress(playerId, temporaryWallet.GetAddrHex()),
		temporaryWallet:       temporaryWallet,
		evtChan:               make(chan *proto.Event, 10),
		errChan:               make(chan error, 10),
		bindOpts:              bindOpts,
		chainClient:           chainClient,
		rpcClient:             rpcClient,
		commitmentOnChainChan: make(chan struct{}),
		roomReadyChan:         make(chan struct{}),
		cardOnChainChan:       make(chan struct{}),
	}, nil
}

func (c *GameContext) Run() error {
	err := c.rpcClient.PubSubClient.Subscribe(c.myself.String(), c.myself.String(), c.evtChan, c.errChan)
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
					phase, err := c.rpcClient.RpcClient.GetGamePhase(c.ctx, c.myself)
					if err != nil {
						fmt.Println("error: ", err.Error())
						continue
					}
					c.lock.Lock()
					c.gameID = uint(phase.GameID)
					for _, pp := range phase.Players {
						player := types.NewPlayerAddress(pp.Address.Id, pp.Address.TemporaryAddress)
						c.players = append(c.players, player)
					}
					c.state = playerStateWaittingConfirm
					c.currentRound = phase.RoundNumber
					c.currentTurn = phase.TurnNumber
					c.lock.Unlock()
				case proto.EventType_TYPE_PART_CONFIRMED:
					fmt.Println("player part confirmed")
				case proto.EventType_TYPE_GAME_CREATED:
					fmt.Println("game created, please submit cards")
					// get contract
					phase, err := c.rpcClient.RpcClient.GetGamePhase(c.ctx, c.myself)
					if err != nil {
						fmt.Println("error: ", err.Error())
					}
					c.lock.Lock()
					c.gameID = uint(phase.GameID)
					c.currentRound = phase.RoundNumber
					c.currentTurn = phase.TurnNumber
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
				case proto.EventType_TYPE_TURN_COMPLETE:
					turnCompleted := evt.GetTurnCompleted()
					if turnCompleted == nil {
						continue
					}
					fmt.Println("turn complete", "round", turnCompleted.RoundNum, "turn", turnCompleted.TurnNum)
					c.lock.Lock()
					c.currentTurn = turnCompleted.TurnNum
					c.lock.Unlock()

					// Handle round completion
					if turnCompleted.IsRoundComplete {
						fmt.Println("round complete, please confirm to start a new round")
						battleInfo, err := c.rpcClient.RpcClient.GetBattleInfo(c.ctx, c.gameID, uint(turnCompleted.RoundNum))
						if err != nil {
							fmt.Println("error: ", err.Error())
						} else {
							fmt.Println("round result: ", types.ToJsonLoggable(battleInfo.RoundResult))
							if !battleInfo.RoundResult.IsGameOver {
								c.lock.Lock()
								c.currentRound++
								c.state = playerStateWaittingConfirm
								c.lock.Unlock()
							} else if turnCompleted.GameResult != nil {
								fmt.Println("game result: ", types.ToJsonLoggable(turnCompleted.GameResult))
							}
						}
					}

					// Handle game completion
					if turnCompleted.IsGameComplete {
						fmt.Println("game complete")
						if turnCompleted.GameResult != nil {
							fmt.Println("game result: ", types.ToJsonLoggable(turnCompleted.GameResult))
						}
						c.lock.Lock()
						c.state = playerStateIdle
						c.lock.Unlock()
					}
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

	err := c.rpcClient.RpcClient.JoinQueue(c.ctx, c.myself)
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
	err := c.rpcClient.RpcClient.ExitQueue(c.ctx, c.myself)
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
	err := c.rpcClient.RpcClient.ConfirmBattle(c.ctx, c.myself, c.gameID, uint(c.currentRound), uint(c.currentTurn))
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
	err := c.rpcClient.RpcClient.ContinueGame(c.ctx, c.myself, c.gameID)
	if err != nil {
		return err
	}
	c.state = playerStateWaittingGameReady
	return nil
}

func (c *GameContext) WaitCommitmentOnChain() {
	<-c.commitmentOnChainChan
}
