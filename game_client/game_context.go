package gameclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/utils"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/crypto"
)

const maxRecoverCardID = uint32(5) // valid card ids are 1..5

// deriveBotSalt returns keccak256(abi.encodePacked(privateKey, gameID, round, turn)).
func deriveBotSalt(w *wallet.Wallet, gameID int64, round, turn uint32) ([]byte, error) {
	hash, err := utils.SolidityPackedKeccak256(
		[]any{
			crypto.FromECDSA(w.GetPrivateKey()),
			gameID,
			round,
			turn,
		},
	)
	if err != nil {
		return nil, err
	}
	return hash.Bytes(), nil
}

// CardPickContext carries everything needed to choose a card for the current turn.
type CardPickContext struct {
	GameID          int64
	Round           uint32
	Turn            uint32
	OpponentID      int64
	OpponentAddress string // temporary address, lowercased (same convention as types.PlayerAddress)
}

// CardProvider is an interface for getting the card to play for a turn
type CardProvider interface {
	GetCard(ctx CardPickContext) (uint32, error)
}

// DeriveRoundThreeCards shuffles [1,2,3,4,5] using keccak256(abi.encodePacked(privateKey, gameID, round))
// as the RNG seed, then returns the first three cards.
func DeriveRoundThreeCards(w *wallet.Wallet, gameID int64, round uint32) ([]uint32, error) {
	if w == nil {
		return nil, fmt.Errorf("wallet is nil")
	}
	seedHash, err := utils.SolidityPackedKeccak256(
		[]any{
			crypto.FromECDSA(w.GetPrivateKey()),
			gameID,
			round,
		},
	)
	if err != nil {
		return nil, err
	}
	var seed [32]byte
	copy(seed[:], seedHash.Bytes())
	rng := rand.New(rand.NewChaCha8(seed))
	deck := []uint32{1, 2, 3, 4, 5}
	rng.Shuffle(5, func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return []uint32{deck[0], deck[1], deck[2]}, nil
}

func opponentForPlayer(myself *types.PlayerAddress, players []*types.PlayerAddress) (id int64, addr string) {
	if myself == nil {
		return 0, ""
	}
	for _, p := range players {
		if p == nil {
			continue
		}
		if p.Id == myself.Id {
			continue
		}
		return p.Id, p.TemporaryAddress
	}
	return 0, ""
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

	gameID       int64
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
	err := c.rpcClient.PubSubClient.Subscribe(subId, c.myself.ToProto(), c.evtChan, c.errChan)
	if err != nil {
		return err
	}
	return nil
}

// SyncGamePhaseIfInGame asks the room server to republish game phase on the event stream only when
// the lobby reports this player is in an active game.
func (c *GameContext) SyncGamePhaseIfInGame() error {
	st, err := c.rpcClient.GetPlayerStatus(c.ctx, c.myself)
	if err != nil {
		return fmt.Errorf("get player status: %w", err)
	}
	if st == nil || st.GetStatus() != proto.PlayerStatus_PLAYER_IN_GAME {
		return nil
	}
	if err := c.rpcClient.SyncGamePhase(c.ctx, c.myself); err != nil {
		return fmt.Errorf("sync game phase: %w", err)
	}
	log.Infow("sync_game_phase", "addr", c.myself.String())
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
				if matched.LastGameId != nil {
					log.Infow("game matched (continue rematch)", "match id", matched.GetMatchId(), "last game id", matched.GetLastGameId())
					if matched.GetMatchId() != 0 {
						if err := c.cancelMatch(matched.GetMatchId()); err != nil {
							log.Errorw("failed to cancel unexpected continue rematch", "match id", matched.GetMatchId(), "error", err)
						} else {
							log.Infow("continue rematch canceled for bot", "match id", matched.GetMatchId())
						}
					}
					continue
				} else {
					log.Infow("game matched", "match id", matched.GetMatchId())
				}
				c.players = c.players[:0] // Reuse slice, more efficient than nil
				for _, pp := range matched.Players {
					c.players = append(c.players, types.NewPlayerAddress(pp.Id, pp.TemporaryAddress))
				}
				c.currentRound = 1
				c.currentTurn = 1
				if matched.GetMatchId() != 0 {
					if err := c.confirmMatch(matched.GetMatchId()); err != nil {
						log.Errorw("failed to confirm match", "error", err)
					}
				}
			case proto.EventType_TYPE_PART_CONFIRMED:
				log.Infow("player part confirmed")
			case proto.EventType_TYPE_GAME_CREATED:
				gr := evt.GetGameReady()
				if gr == nil {
					log.Warnw("game created event missing GameReady data")
					continue
				}
				c.gameID = gr.GetGameId()
				log.Infow("game created", "game id", c.gameID)
				c.players = c.players[:0] // Reuse slice, more efficient than nil
				for _, pp := range gr.Players {
					c.players = append(c.players, types.NewPlayerAddress(pp.Id, pp.TemporaryAddress))
				}
				c.currentRound = 1
				c.currentTurn = 1
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

				oppID, oppAddr := opponentForPlayer(c.myself, c.players)
				card, err := c.cardProvider.GetCard(CardPickContext{
					GameID:          c.gameID,
					Round:           c.currentRound,
					Turn:            c.currentTurn,
					OpponentID:      oppID,
					OpponentAddress: oppAddr,
				})
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
			case proto.EventType_TYPE_GAME_PHASE_SYNC:
				gp := evt.GetGamePhase()
				if gp == nil {
					log.Warnw("game phase sync event missing GamePhase data")
					continue
				}
				log.Infow("game phase sync",
					"game id", gp.GetGameID(),
					"round", gp.GetRoundNumber(),
					"turn", gp.GetTurnNumber(),
					"turn status", gp.GetTurnStatus().String(),
					"player turn status", gp.GetPlayerTurnStatus().String(),
				)
				c.applyGamePhaseRecovery(gp)

			case proto.EventType_TYPE_GAME_SETTLEMENT_RESULT:
				gsr := evt.GetGameSettlementResult()
				if gsr != nil {
					log.Infow("game settlement result", "game id", gsr.GetGameId(), "system_fee", gsr.GetSystemFee(), "player_rewards", types.ToJsonLoggable(gsr.GetPlayerRewards()))
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

func (c *GameContext) confirmMatch(matchID int64) error {
	return c.rpcClient.RpcClient.ConfirmMatch(c.ctx, c.myself, matchID)
}

func (c *GameContext) cancelMatch(matchID int64) error {
	return c.rpcClient.RpcClient.CancelMatch(c.ctx, c.myself, matchID)
}

// PrepareNewCard derives salt from the wallet key and game position, then calculates the commitment for the given card
func (c *GameContext) prepareNewCard(card uint32) error {
	if c.gameID == 0 {
		return fmt.Errorf("game id not set")
	}
	saltBytes, err := deriveBotSalt(c.wallet, c.gameID, c.currentRound, c.currentTurn)
	if err != nil {
		return fmt.Errorf("failed to derive salt: %w", err)
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

// applyGamePhaseRecovery reconciles local state from SyncGamePhase and performs the next RPC if the
// server snapshot shows we are behind (battle confirm, commitment, or card reveal).
func (c *GameContext) applyGamePhaseRecovery(gp *proto.GamePhase) {
	if gp.GetGameID() == 0 {
		return
	}
	c.gameID = gp.GetGameID()
	c.currentRound = gp.GetRoundNumber()
	c.currentTurn = gp.GetTurnNumber()

	if players := gp.GetPlayers(); len(players) > 0 {
		c.players = c.players[:0]
		for _, p := range players {
			if p == nil || p.Address == nil {
				continue
			}
			c.players = append(c.players, types.NewPlayerAddress(p.Address.Id, p.Address.TemporaryAddress))
		}
	}

	ts := gp.GetTurnStatus()
	ps := gp.GetPlayerTurnStatus()

	switch {
	case ts == proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION && ps == proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN:
		if err := c.confirmBattle(); err != nil {
			log.Errorw("game phase recovery: confirm battle failed", "error", err, "game id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
			return
		}
		log.Infow("game phase recovery: confirm battle submitted", "game id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
	case ts == proto.TurnStatus_TURN_WAITTING_COMMITMENTS && ps == proto.PlayerTurnStatus_PLAYER_TURN_READY:
		if c.cardProvider == nil {
			log.Warnw("game phase recovery: card provider not set, skipping commitment", "game id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
			return
		}
		oppID, oppAddr := opponentForPlayer(c.myself, c.players)
		card, err := c.cardProvider.GetCard(CardPickContext{
			GameID:          c.gameID,
			Round:           c.currentRound,
			Turn:            c.currentTurn,
			OpponentID:      oppID,
			OpponentAddress: oppAddr,
		})
		if err != nil {
			log.Errorw("game phase recovery: get card failed", "error", err, "round", c.currentRound, "turn", c.currentTurn)
			return
		}
		if err := c.prepareNewCard(card); err != nil {
			log.Errorw("game phase recovery: prepare new card failed", "error", err, "round", c.currentRound, "turn", c.currentTurn)
			return
		}
		if err := c.submitCommitment(c.commitment); err != nil {
			log.Errorw("game phase recovery: submit commitment failed", "error", err, "round", c.currentRound, "turn", c.currentTurn)
			return
		}
		log.Infow("game phase recovery: commitment submitted", "round", c.currentRound, "turn", c.currentTurn)
	case ts == proto.TurnStatus_TURN_WAITTING_CARDS && ps == proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED:
		self := c.gamePhaseSelf(gp)
		if self == nil {
			log.Warnw("game phase recovery: missing self in GamePhase.Players for card submit", "game id", c.gameID)
			return
		}
		cpi := turnCardPlayingInfoForTurn(self.GetTurnCardPlayingInfos(), gp.GetTurnNumber())
		if err := c.recoverAndSubmitCardFromPhase(cpi); err != nil {
			log.Errorw("game phase recovery: submit card failed", "error", err, "game id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
			return
		}
		log.Infow("game phase recovery: card submitted", "round", c.currentRound, "turn", c.currentTurn)
	default:
		log.Debugw("game phase recovery: no action for snapshot", "turn status", ts.String(), "player turn status", ps.String())
	}
}

func (c *GameContext) gamePhaseSelf(gp *proto.GamePhase) *proto.GamePhasePlayer {
	for _, p := range gp.GetPlayers() {
		if p == nil || p.Address == nil {
			continue
		}
		if p.Address.Id != c.myself.Id {
			continue
		}
		if !strings.EqualFold(p.Address.TemporaryAddress, c.myself.TemporaryAddress) {
			continue
		}
		return p
	}
	return nil
}

func turnCardPlayingInfoForTurn(infos []*proto.TurnCardPlayingInfo, turn uint32) *proto.TurnCardPlayingInfo {
	for _, cpi := range infos {
		if cpi != nil && cpi.TurnNumber == turn {
			return cpi
		}
	}
	return nil
}

func (c *GameContext) recoverAndSubmitCardFromPhase(cpi *proto.TurnCardPlayingInfo) error {
	if c.gameID == 0 {
		return fmt.Errorf("game id not set")
	}
	if cpi == nil || len(cpi.GetCommitment()) == 0 {
		return fmt.Errorf("missing TurnCardPlayingInfo or commitment for recovery")
	}
	want := cpi.GetCommitment()
	saltBytes, err := deriveBotSalt(c.wallet, c.gameID, c.currentRound, c.currentTurn)
	if err != nil {
		return fmt.Errorf("derive salt: %w", err)
	}
	if card := cpi.GetCard(); card != 0 {
		h, err := c.calculateCommitment(card, saltBytes)
		if err != nil {
			return err
		}
		if bytes.Equal(h, want) {
			c.card = card
			c.salt = string(saltBytes)
			c.commitment = want
			return c.submitCard(c.card, c.salt)
		}
	}
	cardID, err := c.recoverCardIDFromCommitment(want, saltBytes)
	if err != nil {
		return err
	}
	c.card = cardID
	c.salt = string(saltBytes)
	c.commitment = want
	return c.submitCard(c.card, c.salt)
}

func (c *GameContext) recoverCardIDFromCommitment(want []byte, salt []byte) (uint32, error) {
	for card := uint32(1); card <= maxRecoverCardID; card++ {
		h, err := c.calculateCommitment(card, salt)
		if err != nil {
			return 0, err
		}
		if bytes.Equal(h, want) {
			return card, nil
		}
	}
	return 0, fmt.Errorf("no card id in 1..%d matches commitment", maxRecoverCardID)
}
