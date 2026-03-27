package gameclient

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/log"
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

func NewGameContext(ctx context.Context,
	playerId int64,
	temporaryWallet *wallet.Wallet,
	rpcClient *rpc.Client,
	cardProvider CardProvider,
) (*GameContext, error) {
	return &GameContext{
		ctx:          ctx,
		myself:       types.NewPlayerAddress(playerId, temporaryWallet.GetAddrHex()),
		wallet:       temporaryWallet,
		evtChan:      make(chan *proto.Event, 10),
		errChan:      make(chan error, 10),
		rpcClient:    rpcClient,
		cardProvider: cardProvider,
		players:      make([]*types.PlayerAddress, 0, 4), // Pre-allocate with capacity
	}, nil
}

func (c *GameContext) Subscribe(id ...string) error {
	var subId string
	if len(id) > 0 {
		subId = id[0]
	} else {
		subId = c.myself.String()
	}
	err := c.rpcClient.PubSubClient.Subscribe(c.myself.String(), subId, c.evtChan, c.errChan)
	if err != nil {
		return err
	}
	return nil
}

func (c *GameContext) Run() error {
	for {
		select {
		case <-c.ctx.Done():
			return errors.New("context done")
		case err := <-c.errChan:
			log.Errorw("subscribe error", "error", err)
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
					log.Warnw("game matched event missing GameMatched data")
					continue
				}
				log.Infow("game matched", "game id", matched.GameId)
				c.gameID = uint(matched.GameId)
				c.players = c.players[:0] // Reuse slice, more efficient than nil
				for _, pp := range matched.Players {
					c.players = append(c.players, types.NewPlayerAddress(pp.Id, pp.TemporaryAddress))
				}
				c.currentRound = 1
				c.currentTurn = 1
				if err := c.confirmBattle(); err != nil {
					log.Errorw("failed to confirm battle", "error", err)
				}
			case proto.EventType_TYPE_PART_CONFIRMED:
				log.Infow("player part confirmed")
			case proto.EventType_TYPE_GAME_CREATED:
				log.Infow("game created")
			case proto.EventType_TYPE_ROUND_READY:
				roundReady := evt.GetRoundReady()
				if roundReady == nil {
					log.Warnw("round ready event missing RoundReady data")
					continue
				}
				// Validate round number
				expectedRoundNum := c.currentRound
				if roundReady.RoundNum != expectedRoundNum {
					log.Warnw("round number mismatch", "expected", expectedRoundNum, "received", roundReady.RoundNum)
					continue
				}
				c.currentRound = roundReady.RoundNum
				c.currentTurn = 1 // Reset turn number for new round
			case proto.EventType_TYPE_TURN_READY:
				turnReady := evt.GetTurnReady()
				if turnReady == nil {
					log.Warnw("turn ready event missing TurnReady data")
					continue
				}

				// Validate round and turn numbers using validateRoundTurn
				expectedTurn := c.currentTurn
				if !c.validateRoundTurnInfo(turnReady, expectedTurn) {
					continue
				}
				// Get card from provider and prepare it
				if c.cardProvider == nil {
					log.Warnw("card provider not set, skipping commitment submission")
					continue
				}

				card, err := c.cardProvider.GetCard(c.currentRound, c.currentTurn)
				if err != nil {
					log.Errorw("failed to get card from provider", "error", err, "round", c.currentRound, "turn", c.currentTurn)
					continue
				}

				// Prepare new card (generates salt and calculates commitment)
				if err := c.prepareNewCard(card); err != nil {
					log.Errorw("failed to prepare new card", "error", err, "round", c.currentRound, "turn", c.currentTurn)
					continue
				}

				// Submit commitment
				if err := c.submitCommitment(c.commitment); err != nil {
					log.Errorw("failed to submit commitment", "error", err, "round", c.currentRound, "turn", c.currentTurn)
				}
			case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
				commitmentsOnChain := evt.GetCommitmentsOnChain()
				if commitmentsOnChain == nil {
					log.Warnw("commitments on chain event missing CommitmentsOnChain data")
					continue
				}
				// Validate round and turn numbers
				if !c.validateRoundTurnInfo(commitmentsOnChain, c.currentTurn) {
					continue
				}
				log.Infow("commitments on chain", "round", commitmentsOnChain.RoundNum, "turn", commitmentsOnChain.TurnNum)

				// Submit turn card and salt
				if c.card == 0 || c.salt == "" {
					log.Warnw("card or salt not set, skipping card submission", "round", c.currentRound, "turn", c.currentTurn)
					continue
				}

				if err := c.submitCard(c.card, c.salt); err != nil {
					log.Errorw("failed to submit card", "error", err, "round", c.currentRound, "turn", c.currentTurn)
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

				log.Infow("turn complete",
					"game id", c.gameID,
					"round", turnCompleted.RoundNum,
					"turn", turnCompleted.TurnNum,
					"isRoundComplete", turnCompleted.IsRoundComplete,
					"isGameComplete", turnCompleted.IsGameComplete)

				// Handle game completion
				if turnCompleted.IsGameComplete {
					log.Infow("game complete", "game id", c.gameID)
					if turnCompleted.GameResult != nil {
						log.Infow("game result", "result", types.ToJsonLoggable(turnCompleted.GameResult))
					}
					return nil
				}
				// Handle round completion
				if turnCompleted.IsRoundComplete {
					c.currentRound++
					c.currentTurn = 1 // Reset turn number for new round
				} else {
					// Turn complete but round not complete - increment turn number and confirm battle for next turn
					c.currentTurn++
				}
				log.Infow("turn complete", "turn completed info", types.ToJsonLoggable(turnCompleted))
				// Confirm battle for the next turn
				if err := c.confirmBattle(); err != nil {
					if turnCompleted.GameResult != nil {
						log.Infow("game result", "result", types.ToJsonLoggable(turnCompleted.GameResult))
					}
					log.Errorw("failed to confirm battle for next turn", "error", err, "round", c.currentRound, "turn", c.currentTurn)
				} else {
					log.Infow("battle confirmed", "round", c.currentRound, "turn", c.currentTurn)
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
		log.Warnw("round number mismatch", "expected", c.currentRound, "received", receivedRound)
		return false
	}
	if receivedTurn != expectedTurn {
		log.Warnw("turn number mismatch", "expected", expectedTurn, "received", receivedTurn)
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
	log.Infow("submitting commitment", "round", c.currentRound, "turn", c.currentTurn)
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
	log.Infow("commitment submitted successfully", "round", c.currentRound, "turn", c.currentTurn)
	return nil
}

// submitCard submits the card and salt via RPC
func (c *GameContext) submitCard(card uint32, salt string) error {
	log.Infow("submitting card", "round", c.currentRound, "turn", c.currentTurn)
	// Generate signature: game id, round number, card index, card, salt
	signature, err := utils.Sign(
		[]any{
			c.gameID,
			c.currentRound,
			c.currentTurn,
			card,
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
	log.Infow("card submitted successfully", "round", c.currentRound, "turn", c.currentTurn)
	return nil
}

func (c *GameContext) Continue() error {
	err := c.rpcClient.RpcClient.ContinueGame(c.ctx, c.myself, c.gameID)
	if err != nil {
		return err
	}
	return nil
}
