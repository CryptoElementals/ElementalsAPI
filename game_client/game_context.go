package gameclient

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/utils"
	"github.com/CryptoElementals/common/wallet"
)

const (
	saltSize = 32
)

// CardProvider is an interface for getting the card to play for a turn
type CardProvider interface {
	GetCard(round uint32, turn uint32) (uint32, error)
}

// RoundTurnInfo is an interface for event types that have round and turn numbers
type RoundTurnInfo interface {
	GetRoundNum() uint32
	GetTurnNum() uint32
}

type GameContext struct {
	ctx          context.Context
	rpcClient    *rpc.Client
	myself       *types.PlayerAddress
	wallet       *wallet.Wallet
	evtChan      chan *proto.Event
	errChan      chan error
	cardProvider CardProvider // Interface to get card for each turn

	gameID       uint
	players      []*types.PlayerAddress
	currentRound uint32
	currentTurn  uint32
	card         uint32
	salt         string
	commitment   []byte
}

func NewGameContext(ctx context.Context, playerId int64, temporaryWallet *wallet.Wallet, rpcClient *rpc.Client) (*GameContext, error) {
	return &GameContext{
		ctx:       ctx,
		myself:    types.NewPlayerAddress(playerId, temporaryWallet.GetAddrHex()),
		wallet:    temporaryWallet,
		evtChan:   make(chan *proto.Event, 10),
		errChan:   make(chan error, 10),
		rpcClient: rpcClient,
		players:   make([]*types.PlayerAddress, 0, 4), // Pre-allocate with capacity
	}, nil
}

// SetCardProvider sets the card provider interface for getting cards
func (c *GameContext) SetCardProvider(provider CardProvider) {
	c.cardProvider = provider
}

func (c *GameContext) Run() error {
	err := c.rpcClient.PubSubClient.Subscribe(c.myself.String(), c.myself.String(), c.evtChan, c.errChan)
	if err != nil {
		return err
	}
	err = c.JoinQueue()
	if err != nil {
		return err
	}
	for {
		select {
		case <-c.ctx.Done():
			return errors.New("context done")
		case err := <-c.errChan:
			fmt.Println("subscribe err: ", err)
		case evt, ok := <-c.evtChan:
			if !ok {
				return errors.New("event channel closed")
			}
			switch evt.Type {
			case proto.EventType_TYPE_KNOWN:
				return errors.New("event known")
			case proto.EventType_TYPE_MATCHED:
				matched := evt.GetGameMatched()
				if matched == nil {
					fmt.Println("game matched event missing GameMatched data")
					continue
				}
				fmt.Println("game matched", "game id", matched.GameId)
				c.gameID = uint(matched.GameId)
				c.players = c.players[:0] // Reuse slice, more efficient than nil
				for _, pp := range matched.Players {
					c.players = append(c.players, types.NewPlayerAddress(pp.Id, pp.TemporaryAddress))
				}
				c.currentRound = 1
				c.currentTurn = 1
				if err := c.confirmBattle(); err != nil {
					fmt.Println("error: ", err.Error())
				}
			case proto.EventType_TYPE_PART_CONFIRMED:
				fmt.Println("player part confirmed")
			case proto.EventType_TYPE_GAME_CREATED:
				fmt.Println("game created")
			case proto.EventType_TYPE_ROUND_READY:
				roundReady := evt.GetRoundReady()
				if roundReady == nil {
					fmt.Println("round ready event missing RoundReady data")
					continue
				}
				// Validate round number
				expectedRoundNum := c.currentRound
				if roundReady.RoundNum != expectedRoundNum {
					fmt.Printf("round number mismatch: expected %d, received %d\n", expectedRoundNum, roundReady.RoundNum)
					continue
				}
				c.currentRound = roundReady.RoundNum
				c.currentTurn = 1 // Reset turn number for new round
			case proto.EventType_TYPE_TURN_READY:
				turnReady := evt.GetTurnReady()
				if turnReady == nil {
					fmt.Println("turn ready event missing TurnReady data")
					continue
				}

				// Validate round and turn numbers using validateRoundTurn
				expectedTurn := c.currentTurn
				if !c.validateRoundTurnInfo(turnReady, expectedTurn) {
					continue
				}
				// Get card from provider and prepare it
				if c.cardProvider == nil {
					fmt.Println("Warning: card provider not set, skipping commitment submission")
					continue
				}

				card, err := c.cardProvider.GetCard(c.currentRound, c.currentTurn)
				if err != nil {
					fmt.Printf("Failed to get card from provider: %v\n", err)
					continue
				}

				// Prepare new card (generates salt and calculates commitment)
				if err := c.prepareNewCard(card); err != nil {
					fmt.Printf("Failed to prepare new card: %v\n", err)
					continue
				}

				// Submit commitment
				if err := c.submitCommitment(c.commitment); err != nil {
					fmt.Printf("Failed to submit commitment: %v\n", err)
				}
			case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
				commitmentsOnChain := evt.GetCommitmentsOnChain()
				if commitmentsOnChain == nil {
					fmt.Println("commitments on chain event missing CommitmentsOnChain data")
					continue
				}
				// Validate round and turn numbers
				if !c.validateRoundTurnInfo(commitmentsOnChain, c.currentTurn) {
					continue
				}
				fmt.Println("commitments on chain", "round", commitmentsOnChain.RoundNum, "turn", commitmentsOnChain.TurnNum)

				// Submit turn card and salt
				if c.card == 0 || c.salt == "" {
					fmt.Println("Warning: card or salt not set, skipping card submission")
					continue
				}

				if err := c.submitCard(c.card, c.salt); err != nil {
					fmt.Printf("Failed to submit card: %v\n", err)
				}
			case proto.EventType_TYPE_TURN_COMPLETE:
				turnCompleted := evt.GetTurnCompleted()
				if turnCompleted == nil {
					continue
				}
				// Validate round and turn numbers
				if !c.validateRoundTurnInfo(turnCompleted, c.currentTurn) {
					continue
				}

				fmt.Println("turn complete", "round", turnCompleted.RoundNum, "turn", turnCompleted.TurnNum, "isRoundComplete", turnCompleted.IsRoundComplete, "isGameComplete", turnCompleted.IsGameComplete)

				// Handle game completion
				if turnCompleted.IsGameComplete {
					fmt.Println("game complete")
					if turnCompleted.GameResult != nil {
						fmt.Println("game result: ", types.ToJsonLoggable(turnCompleted.GameResult))
					}
					return nil
				}

				// Handle round completion
				if turnCompleted.IsRoundComplete {
					battleInfo, err := c.rpcClient.RpcClient.GetBattleInfo(c.ctx, c.gameID, uint(turnCompleted.RoundNum))
					if err != nil {
						fmt.Println("error: ", err.Error())
					} else {
						fmt.Println("round result: ", types.ToJsonLoggable(battleInfo.RoundResult))
					}
					c.currentRound++
					c.currentTurn = 1 // Reset turn number for new round
				} else {
					// Turn complete but round not complete - increment turn number and confirm battle for next turn
					c.currentTurn++
				}
				// Confirm battle for the next turn
				if err := c.confirmBattle(); err != nil {
					fmt.Printf("Failed to confirm battle for next turn: %v\n", err)
				} else {
					fmt.Printf("Battle confirmed for round %d, turn %d\n", c.currentRound, c.currentTurn)
				}
			}
		}
	}
}

func (c *GameContext) JoinQueue() error {
	err := c.rpcClient.RpcClient.JoinQueue(c.ctx, c.myself)
	if err != nil {
		return err
	}
	return nil
}

func (c *GameContext) ExitQueue() error {
	err := c.rpcClient.RpcClient.ExitQueue(c.ctx, c.myself)
	if err != nil {
		return err
	}
	return nil
}

func (c *GameContext) confirmBattle() error {
	err := c.rpcClient.RpcClient.ConfirmBattle(c.ctx, c.myself, c.gameID, uint(c.currentRound), uint(c.currentTurn))
	if err != nil {
		return err
	}
	return nil
}

// PrepareNewCard generates a new salt and calculates the commitment for the given card
func (c *GameContext) prepareNewCard(card uint32) error {
	// Generate random bytes for salt
	saltBytes := make([]byte, saltSize)
	if _, err := rand.Read(saltBytes); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Store card and salt
	c.card = card
	c.salt = string(saltBytes)

	// Calculate commitment: Keccak256(cardString + saltBytes)
	commitment, err := c.calculateCommitment(card, saltBytes)
	if err != nil {
		return fmt.Errorf("failed to calculate commitment: %w", err)
	}
	c.commitment = commitment

	return nil
}

// calculateCommitment calculates the Keccak256 hash of card string + salt bytes
func (c *GameContext) calculateCommitment(card uint32, saltBytes []byte) ([]byte, error) {
	hash, err := utils.SolidityPackedKeccak256(
		[]any{
			card,
			saltBytes,
		},
	)
	if err != nil {
		return nil, err
	}
	return hash.Bytes(), nil
}

// validateRoundTurn validates that the received round and turn numbers match expected values
func (c *GameContext) validateRoundTurn(receivedRound, receivedTurn, expectedTurn uint32) bool {
	if receivedRound != c.currentRound {
		fmt.Printf("round number mismatch: expected %d, received %d\n", c.currentRound, receivedRound)
		return false
	}
	if receivedTurn != expectedTurn {
		fmt.Printf("turn number mismatch: expected %d, received %d\n", expectedTurn, receivedTurn)
		return false
	}
	return true
}

// validateRoundTurnInfo validates round and turn numbers from a RoundTurnInfo interface
func (c *GameContext) validateRoundTurnInfo(info RoundTurnInfo, expectedTurn uint32) bool {
	return c.validateRoundTurn(info.GetRoundNum(), info.GetTurnNum(), expectedTurn)
}

// submitCommitment submits the commitment via RPC
func (c *GameContext) submitCommitment(commitment []byte) error {
	fmt.Println("submit commitment, round: ", c.currentRound)
	// Generate signature: game id, round number, commitment index, commitment
	signature, err := utils.Sign(
		[]any{
			c.gameID,
			c.currentRound,
			c.currentTurn,
			commitment,
		},
		c.wallet.GetPrivateKey(),
	)
	if err != nil {
		return fmt.Errorf("failed to generate signature: %w", err)
	}
	err = c.rpcClient.RpcClient.SubmitPlayerCommitment(
		c.ctx,
		c.myself,
		c.currentRound,
		commitment,
		c.currentTurn,
		signature,
		c.gameID,
	)
	if err != nil {
		return err
	}
	fmt.Printf("Commitment submitted successfully for round %d, turn %d\n", c.currentRound, c.currentTurn)
	return nil
}

// submitCard submits the card and salt via RPC
func (c *GameContext) submitCard(card uint32, salt string) error {
	cardStr := fmt.Sprintf("%d", card)
	fmt.Println("submit cards, round: ", c.currentRound)
	// Generate signature: game id, round number, card index, card, salt
	signature, err := utils.Sign(
		[]any{
			c.gameID,
			c.currentRound,
			c.currentTurn,
			cardStr,
			salt,
		},
		c.wallet.GetPrivateKey(),
	)
	if err != nil {
		return fmt.Errorf("failed to generate signature: %w", err)
	}
	err = c.rpcClient.RpcClient.SubmitPlayerCard(
		c.ctx,
		c.myself,
		c.currentRound,
		[]byte(salt),
		uint(card),
		c.currentTurn,
		signature,
		c.gameID,
	)
	if err != nil {
		return err
	}
	fmt.Printf("Card submitted successfully for round %d, turn %d\n", c.currentRound, c.currentTurn)
	return nil
}

func (c *GameContext) Continue() error {
	err := c.rpcClient.RpcClient.ContinueGame(c.ctx, c.myself, c.gameID)
	if err != nil {
		return err
	}
	return nil
}
