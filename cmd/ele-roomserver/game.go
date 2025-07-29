/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/c-bata/go-prompt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

// gameCmd represents the game command
var gameCmd = &cobra.Command{
	Use:   "game",
	Short: "game tools for room server testing",
	Run: func(cmd *cobra.Command, args []string) {
		err := startGame()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

var chainRpc string
var roomServerEndpoint string
var walletPath string

func init() {
	rootCmd.AddCommand(gameCmd)
	gameCmd.Flags().StringVarP(&chainRpc, "chain-rpc", "c", "", "chain rpc endpoint")
	gameCmd.Flags().StringVarP(&roomServerEndpoint, "room-server-endpoint", "r", "", "room server endpoint")
	gameCmd.Flags().StringVarP(&walletPath, "wallet-path", "w", "", "wallet path")
	gameCmd.MarkFlagRequired("chain-rpc")
	gameCmd.MarkFlagRequired("room-server-endpoint")
	gameCmd.MarkFlagRequired("wallet-path")
}

func toJsonLoggable(obj any) string {
	res, _ := json.MarshalIndent(obj, "", "  ")
	return string(res)
}

var suggestions = []prompt.Suggest{
	{Text: "join-queue", Description: "join game queue"},
	{Text: "exit-queue", Description: "exit game queue"},
	{Text: "confirm", Description: "confirm game"},
	//{Text: "submit-commitment", Description: "submit card commitments"},
	{Text: "submit-cards", Description: "submit cards"},
	{Text: "continue", Description: "continue game"},
}

func completer(in prompt.Document) []prompt.Suggest {
	w := in.GetWordBeforeCursor()
	if w == "" {
		return []prompt.Suggest{}
	}
	return prompt.FilterHasPrefix(suggestions, w, true)
}

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

type gameContext struct {
	ctx             context.Context
	wallet          *wallet.Wallet
	temporaryWallet *wallet.Wallet
	chainClient     *ethclient.Client
	rpcClient       *rpc.Client

	gameID          uint
	players         []*types.PlayerAddress
	currentRound    uint32
	contractAddress string
	myself          *types.PlayerAddress
	contract        *contract.RoomContract
	bindOpts        *bind.TransactOpts
	evtChan         chan *proto.Event
	errChan         chan error
	state           playerState
	lock            sync.Mutex
	signalChan      chan struct{}
}

func newGameContext(ctx context.Context, wallet *wallet.Wallet, temporaryWallet *wallet.Wallet, chainClient *ethclient.Client, rpcClient *rpc.Client) (*gameContext, error) {
	chainId, err := chainClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	temporaryWallet.BuildTxSinger(chainId)
	bindOpts := &bind.TransactOpts{
		From:   temporaryWallet.GetAddr(),
		Signer: temporaryWallet.BuildTxSinger(chainId),
	}
	return &gameContext{
		ctx:             ctx,
		myself:          types.NewPlayerAddress(wallet.GetAddrHex(), temporaryWallet.GetAddrHex()),
		wallet:          wallet,
		temporaryWallet: temporaryWallet,
		evtChan:         make(chan *proto.Event, 10),
		errChan:         make(chan error, 10),
		bindOpts:        bindOpts,
		chainClient:     chainClient,
		rpcClient:       rpcClient,
		signalChan:      make(chan struct{}),
	}, nil
}

func (c *gameContext) run() error {
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
					c.gameID = uint(phase.PvPInfo.GameID)
					for _, pp := range phase.Players {
						player := types.NewPlayerAddress(pp.Address.WalletAddress, pp.Address.TemporaryAddress)
						c.players = append(c.players, player)
					}
					c.state = playerStateWaittingConfirm
					c.currentRound = 1
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
					c.gameID = uint(phase.PvPInfo.GameID)
					c.contractAddress = phase.PvPInfo.ContractAddress
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
					c.signalChan <- struct{}{}
				case proto.EventType_TYPE_CARDS_ON_CHAIN:
					c.lock.Lock()
					fmt.Println("cards on chain")
					c.state = playerStateWaittingRoundEnd
					c.lock.Unlock()
				case proto.EventType_TYPE_ROUND_COMPLETE:
					fmt.Println("round complete, please confirm to start a new round")
					battleInfo, err := c.rpcClient.RpcClient.GetBattleInfo(c.ctx, c.gameID, uint(c.currentRound))
					if err != nil {
						fmt.Println("error: ", err.Error())
					}
					fmt.Println("round result: ", toJsonLoggable(battleInfo.RoundResult))
					if !battleInfo.RoundResult.IsGameOver {
						c.lock.Lock()
						c.currentRound++
						c.state = playerStateWaittingConfirm
						c.lock.Unlock()
					} else {
						fmt.Println("game result: ", toJsonLoggable(battleInfo.GameResult))
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

func (c *gameContext) JoinQueue() error {
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

func (c *gameContext) ExitQueue() error {
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

func (c *gameContext) ConfirmBattle() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.state != playerStateWaittingConfirm {
		return fmt.Errorf("cannot confirm battle, invalid state: %s", c.state.String())
	}
	err := c.rpcClient.RpcClient.ConfirmBattle(c.ctx, c.myself, c.gameID, uint(c.currentRound))
	if err != nil {
		return err
	}
	c.state = playerStateWaittingGameReady
	return nil
}

func (c *gameContext) SubmitCommitment(cardHash []byte) (string, error) {
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

func (c *gameContext) SubmitCards(cards string, salt string) (string, error) {
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

func (c *gameContext) Continue() error {
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

func makeExecutor(ctx *gameContext) func(in string) {
	return func(in string) {
		in = strings.TrimSpace(in)
		blocks := strings.Split(in, " ")
		cmd := blocks[0]
		blocks = blocks[1:]
		assetArgsNumber := func(expectedCount int) error {
			if len(blocks) != expectedCount {
				return fmt.Errorf("invalid args number")
			}
			return nil
		}
		switch cmd {
		case "join-queue":
			err := assetArgsNumber(0)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = ctx.JoinQueue()
			if err != nil {
				fmt.Println("join queue failed, err: ", err)
				return
			}
			fmt.Println("join queue success, waitting for game matched")
		case "exit-queue":
			err := assetArgsNumber(0)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = ctx.ExitQueue()
			if err != nil {
				fmt.Println("exit queue failed, err: ", err)
				return
			}
			fmt.Println("exit queue success")
		case "confirm":
			err := assetArgsNumber(0)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = ctx.ConfirmBattle()
			if err != nil {
				fmt.Println("confirm failed, err: ", err)
				return
			}
			fmt.Println("confirm success, waitting for game ready on chain")
		case "continue":
			err := assetArgsNumber(0)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = ctx.Continue()
			if err != nil {
				fmt.Println("continue failed, err: ", err)
				return
			}
			fmt.Println("continue success, waitting for game ready on chain")
		case "submit-cards":
			err := assetArgsNumber(3)
			if err != nil {
				fmt.Println(err)
				return
			}
			cards := strings.Join(blocks[:3], ",")
			salt := "salt"
			hh := sha3.NewLegacyKeccak256()
			hh.Write([]byte(cards))
			hh.Write([]byte(salt))
			commitment := hh.Sum(nil)
			tx, err := ctx.SubmitCommitment(commitment)
			if err != nil {
				fmt.Println("submit commitment failed, err: ", err)
				return
			}
			fmt.Println("submit commitment success, tx hash: ", tx)
			fmt.Println("waitting for commitment on chain")
			<-ctx.signalChan
			tx, err = ctx.SubmitCards(cards, salt)
			if err != nil {
				fmt.Println("submit cards failed, err: ", err)
				return
			}
			fmt.Println("submit cards success, tx hash: ", tx)
		}
	}
}

func startGame() error {
	if chainRpc == "" || roomServerEndpoint == "" {
		return fmt.Errorf("chain rpc and room server endpoint are required")
	}
	client, err := rpc.NewClient(roomServerEndpoint)
	if err != nil {
		return err
	}
	chainClient, err := ethclient.Dial(chainRpc)
	if err != nil {
		return err
	}
	var wTemp *wallet.Wallet

	wTemp, err = wallet.LoadWallet(walletPath)
	if err != nil {
		return err
	}
	fmt.Println("using temp account, address: ", wTemp.GetAddrHex())
	w, err := wallet.NewWallet("")
	if err != nil {
		return err
	}
	fmt.Println("using generated account, address: ", w.GetAddrHex())
	gameContext, err := newGameContext(context.Background(), w, wTemp, chainClient, client)
	if err != nil {
		return err
	}
	err = gameContext.run()
	if err != nil {
		return err
	}
	executor := makeExecutor(gameContext)
	p := prompt.New(executor, completer)
	p.Run()

	return nil
}
