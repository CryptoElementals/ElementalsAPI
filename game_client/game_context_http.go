package gameclient

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/utils"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// GameContextHTTP is an HTTP-based game client for stress testing
type GameContextHTTP struct {
	ctx          context.Context
	apiClient    *HttpApiClient
	wallet       *wallet.Wallet
	cardProvider CardProvider

	gameID       int64
	currentRound uint32
	currentTurn  uint32
	card         uint32
	salt         []byte
	commitment   []byte

	// Session management
	playerID     string
	address      string
	refreshToken string

	// SSE connection
	eventChan <-chan *proto.Event
	errChan   <-chan error
	sseCancel context.CancelFunc
}

// NewGameContextHTTP creates a new HTTP-based game context
func NewGameContextHTTP(
	ctx context.Context,
	baseURL string,
	playerId int64,
	temporaryWallet *wallet.Wallet,
	cardProvider CardProvider,
) (*GameContextHTTP, error) {
	apiClient, err := NewHttpApiClient(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP API client: %w", err)
	}

	address := strings.ToLower(temporaryWallet.GetAddrHex())
	return &GameContextHTTP{
		ctx:          ctx,
		apiClient:    apiClient,
		wallet:       temporaryWallet,
		cardProvider: cardProvider,
		address:      address,
		playerID:     fmt.Sprintf("%d", playerId), // Will be updated after login
	}, nil
}

// SignIn signs in the user via HTTP API and fetches player ID
func (c *GameContextHTTP) SignIn() error {
	// Step 1: Get login code (nonce)
	nonce, loginCode, err := c.apiClient.GetLoginCode(c.address)
	if err != nil {
		return fmt.Errorf("failed to get login code: %w", err)
	}

	// Step 2: Sign the login code
	signature, err := c.wallet.EthSign(loginCode)
	if err != nil {
		return fmt.Errorf("failed to sign login code: %w", err)
	}
	signatureHex := hexutil.Encode(signature)

	// Step 3: Login with signature and get refresh token
	_, refreshToken, err := c.apiClient.Login(signatureHex, c.address, nonce)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	c.refreshToken = refreshToken

	// Step 4: Get player ID from IsUserLoggedIn API
	playerID, loggedIn, err := c.apiClient.IsUserLoggedIn(refreshToken)
	if err != nil {
		return fmt.Errorf("failed to check login status: %w", err)
	}
	if !loggedIn {
		return fmt.Errorf("user not logged in after login")
	}
	c.playerID = playerID

	log.Infow("signed in successfully", "player_id", c.playerID, "address", c.address)
	return nil
}

// GetRefreshToken returns the refresh token
func (c *GameContextHTTP) GetRefreshToken() string {
	return c.refreshToken
}

// GetPlayerID returns the player ID
func (c *GameContextHTTP) GetPlayerID() string {
	return c.playerID
}

// GetApiClient returns the HTTP API client
func (c *GameContextHTTP) GetApiClient() *HttpApiClient {
	return c.apiClient
}

// Subscribe subscribes to game events via SSE
func (c *GameContextHTTP) Subscribe() error {
	eventChan, errChan, cancel, err := c.apiClient.SubscribeGameInfo(c.address, c.playerID, 86400)
	if err != nil {
		return err
	}
	c.eventChan = eventChan
	c.errChan = errChan
	c.sseCancel = cancel
	return nil
}

// Run runs the game event loop (same logic as gRPC version)
func (c *GameContextHTTP) Run() error {
	for {
		select {
		case <-c.ctx.Done():
			return errors.New("context done")
		case err := <-c.errChan:
			log.Errorw("subscribe error", "player_id", c.playerID, "error", err)
			err = c.Subscribe()
			if err != nil {
				return err
			}
		case evt, ok := <-c.eventChan:
			if !ok {
				return errors.New("event channel closed")
			}
			switch evt.Type {
			case proto.EventType_TYPE_KNOWN:
				return errors.New("event known")
			case proto.EventType_TYPE_MATCHED:
				matched := evt.GetGameMatched()
				if matched == nil {
					log.Warnw("game matched event missing GameMatched data", "player_id", c.playerID)
					continue
				}
				if matched.LastGameId != nil {
					log.Infow("game matched (continue rematch)", "player_id", c.playerID, "match_id", matched.GetMatchId(), "last_game_id", matched.GetLastGameId())
					if matched.GetMatchId() != 0 {
						if err := c.cancelMatch(matched.GetMatchId()); err != nil {
							log.Errorw("failed to cancel unexpected continue rematch", "player_id", c.playerID, "match_id", matched.GetMatchId(), "error", err)
						}
					}
					continue
				} else {
					log.Infow("game matched", "player_id", c.playerID, "match_id", matched.GetMatchId())
				}
				c.currentRound = 1
				c.currentTurn = 1
				if matched.GetMatchId() != 0 {
					if err := c.confirmMatch(matched.GetMatchId()); err != nil {
						log.Errorw("failed to confirm match", "player_id", c.playerID, "match_id", matched.GetMatchId(), "error", err)
					}
				}
			case proto.EventType_TYPE_PART_CONFIRMED:
				log.Infow("player part confirmed", "player_id", c.playerID, "game_id", c.gameID)
			case proto.EventType_TYPE_GAME_CREATED:
				gr := evt.GetGameReady()
				if gr == nil {
					log.Warnw("game created event missing GameReady data", "player_id", c.playerID)
					continue
				}
				c.gameID = gr.GetGameId()
				log.Infow("game created", "player_id", c.playerID, "game_id", c.gameID)
				c.currentRound = 1
				c.currentTurn = 1
			case proto.EventType_TYPE_ROUND_READY:
				roundReady := evt.GetRoundReady()
				if roundReady == nil {
					log.Warnw("round ready event missing RoundReady data", "player_id", c.playerID, "game_id", c.gameID)
					continue
				}
				expectedRoundNum := c.currentRound
				if roundReady.RoundNum != expectedRoundNum {
					log.Warnw("round number mismatch", "player_id", c.playerID, "game_id", c.gameID, "expected", expectedRoundNum, "received", roundReady.RoundNum)
					continue
				}
				c.currentRound = roundReady.RoundNum
				c.currentTurn = 1
				log.Infow("round ready", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound)
			case proto.EventType_TYPE_TURN_READY:
				turnReady := evt.GetTurnReady()
				if turnReady == nil {
					log.Warnw("turn ready event missing TurnReady data", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
					continue
				}

				if !c.validateRoundTurnInfo(turnReady, c.currentTurn) {
					continue
				}

				if c.cardProvider == nil {
					log.Warnw("card provider not set, skipping commitment submission", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
					continue
				}

				card, err := c.cardProvider.GetCard(c.currentRound, c.currentTurn)
				if err != nil {
					log.Errorw("failed to get card from provider", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn, "error", err)
					continue
				}

				if err := c.prepareNewCard(card); err != nil {
					log.Errorw("failed to prepare new card", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn, "error", err)
					continue
				}

				if err := c.submitCommitment(c.commitment); err != nil {
					log.Errorw("failed to submit commitment", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn, "error", err)
				}
			case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
				commitmentsOnChain := evt.GetCommitmentsOnChain()
				if commitmentsOnChain == nil {
					log.Warnw("commitments on chain event missing CommitmentsOnChain data", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
					continue
				}
				if !c.validateRoundTurnInfo(commitmentsOnChain, c.currentTurn) {
					continue
				}
				log.Infow("commitments on chain", "player_id", c.playerID, "game_id", c.gameID, "round", commitmentsOnChain.RoundNum, "turn", commitmentsOnChain.TurnNum)

				if c.card == 0 || len(c.salt) == 0 {
					log.Warnw("card or salt not set, skipping card submission", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
					continue
				}

				if err := c.submitCard(c.card, c.salt); err != nil {
					log.Errorw("failed to submit card", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn, "error", err)
				}
			case proto.EventType_TYPE_TURN_COMPLETE:
				turnCompleted := evt.GetTurnCompleted()
				if turnCompleted == nil {
					continue
				}
				if !c.validateRoundTurnInfo(turnCompleted, c.currentTurn) {
					continue
				}

				log.Infow("turn complete",
					"player_id", c.playerID,
					"game_id", c.gameID,
					"round", turnCompleted.RoundNum,
					"turn", turnCompleted.TurnNum,
					"isRoundComplete", turnCompleted.IsRoundComplete,
					"isGameComplete", turnCompleted.IsGameComplete)

				if turnCompleted.IsGameComplete {
					log.Infow("game complete", "player_id", c.playerID, "game_id", c.gameID)
					if turnCompleted.GameResult != nil {
						log.Infow("game result", "player_id", c.playerID, "game_id", c.gameID, "result", types.ToJsonLoggable(turnCompleted.GameResult))
					}
					return nil
				}
				if turnCompleted.IsRoundComplete {
					c.currentRound++
					c.currentTurn = 1
				} else {
					c.currentTurn++
				}
				log.Infow("turn complete", "player_id", c.playerID, "game_id", c.gameID, "round", turnCompleted.RoundNum, "turn", turnCompleted.TurnNum, "turn_completed_info", types.ToJsonLoggable(turnCompleted))
				if err := c.confirmBattle(); err != nil {
					if turnCompleted.GameResult != nil {
						log.Infow("game result", "player_id", c.playerID, "game_id", c.gameID, "result", types.ToJsonLoggable(turnCompleted.GameResult))
					}
					log.Errorw("failed to confirm battle for next turn", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn, "error", err)
				} else {
					log.Infow("battle confirmed", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
				}
			case proto.EventType_TYPE_GAME_SETTLEMENT_RESULT:
				gsr := evt.GetGameSettlementResult()
				if gsr != nil {
					log.Infow("game settlement result", "player_id", c.playerID, "game_id", gsr.GetGameId(), "system_fee", gsr.GetSystemFee(), "player_rewards", types.ToJsonLoggable(gsr.GetPlayerRewards()))
				}
			}
		}
	}
}

// JoinQueue joins the match queue via HTTP API
func (c *GameContextHTTP) JoinQueue() error {
	err := c.apiClient.JoinQueue("PvP", c.address, c.playerID)
	if err != nil {
		return err
	}
	log.Infow("joined queue successfully", "player_id", c.playerID)
	return nil
}

// ExitQueue exits the match queue via HTTP API
func (c *GameContextHTTP) ExitQueue() error {
	return c.apiClient.ExitQueue(c.address, c.playerID)
}

// confirmBattle confirms battle via HTTP API
func (c *GameContextHTTP) confirmBattle() error {
	return c.apiClient.ConfirmBattle(c.gameID, c.currentRound, c.currentTurn, c.address, c.playerID)
}

func (c *GameContextHTTP) confirmMatch(matchID int64) error {
	return c.apiClient.ConfirmMatch(matchID, c.address, c.playerID)
}

func (c *GameContextHTTP) cancelMatch(matchID int64) error {
	return c.apiClient.CancelMatch(matchID, c.address, c.playerID)
}

// prepareNewCard derives salt from the wallet key and game position, then calculates the commitment for the given card
func (c *GameContextHTTP) prepareNewCard(card uint32) error {
	if c.gameID == 0 {
		return fmt.Errorf("game id not set")
	}
	saltBytes, err := deriveBotSalt(c.wallet, c.gameID, c.currentRound, c.currentTurn)
	if err != nil {
		return fmt.Errorf("failed to derive salt: %w", err)
	}

	c.card = card
	c.salt = saltBytes

	commitment, err := c.calculateCommitment(card, saltBytes)
	if err != nil {
		return fmt.Errorf("failed to calculate commitment: %w", err)
	}
	c.commitment = commitment

	return nil
}

// calculateCommitment calculates the Keccak256 hash of card string + salt bytes
func (c *GameContextHTTP) calculateCommitment(card uint32, saltBytes []byte) ([]byte, error) {
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

// submitCommitment submits the commitment via HTTP API
func (c *GameContextHTTP) submitCommitment(commitment []byte) error {
	log.Infow("submitting commitment", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)

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

	if err := c.apiClient.SubmitPlayerCommitment(
		c.gameID,
		c.currentRound,
		c.currentTurn,
		commitment,
		signature,
		c.address,
		c.playerID,
	); err != nil {
		return fmt.Errorf("failed to submit commitment: %w", err)
	}

	log.Infow("commitment submitted successfully", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
	return nil
}

// submitCard submits the card and salt via HTTP API
func (c *GameContextHTTP) submitCard(card uint32, salt []byte) error {
	log.Infow("submitting card", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)

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

	// Encode salt bytes to hex string for the API
	saltHex := hexutil.Encode(salt)
	if err := c.apiClient.SubmitPlayerCard(
		c.gameID,
		c.currentRound,
		c.currentTurn,
		card,
		saltHex,
		signature,
		c.address,
		c.playerID,
	); err != nil {
		return fmt.Errorf("failed to submit card: %w", err)
	}

	log.Infow("card submitted successfully", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "turn", c.currentTurn)
	return nil
}

// validateRoundTurn validates that the received round and turn numbers match expected values
func (c *GameContextHTTP) validateRoundTurn(receivedRound, receivedTurn, expectedTurn uint32) bool {
	if receivedRound != c.currentRound {
		log.Warnw("round number mismatch", "player_id", c.playerID, "game_id", c.gameID, "expected", c.currentRound, "received", receivedRound)
		return false
	}
	if receivedTurn != expectedTurn {
		log.Warnw("turn number mismatch", "player_id", c.playerID, "game_id", c.gameID, "round", c.currentRound, "expected", expectedTurn, "received", receivedTurn)
		return false
	}
	return true
}

// validateRoundTurnInfo validates round and turn numbers from a RoundTurnInfo interface
func (c *GameContextHTTP) validateRoundTurnInfo(info RoundTurnInfo, expectedTurn uint32) bool {
	return c.validateRoundTurn(info.GetRoundNum(), info.GetTurnNum(), expectedTurn)
}

// Close closes the SSE connection
func (c *GameContextHTTP) Close() {
	if c.sseCancel != nil {
		c.sseCancel()
	}
}
