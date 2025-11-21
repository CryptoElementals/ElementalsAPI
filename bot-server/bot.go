package botserver

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

type playerWallet struct {
	playerId   int64
	tempWallet *wallet.Wallet
}

func (w *playerWallet) address() *types.PlayerAddress {
	return types.NewPlayerAddress(w.playerId, w.tempWallet.GetAddrHex())
}

type roundInfo struct {
	roundNum    uint
	turnNumber  uint32 // Current turn number (1-3)
	commitments [][32]byte
	cards       []uint32 // Store cards as array for easier access
	salts       []string
}

func (i *roundInfo) prepareNewRound() {
	i.roundNum++
	i.turnNumber = 0 // Reset turn number for new round
	i.cards = nil
	i.commitments = nil
	i.salts = nil
}

// prepareCards prepares 3 cards, salts, and commitments for the current round
func (i *roundInfo) prepareCards() {
	// Select random cards
	allCards := make([]uint32, 5)
	for j := range allCards {
		allCards[j] = uint32(j + 1)
	}
	rand.Shuffle(5, func(j, k int) {
		allCards[j], allCards[k] = allCards[k], allCards[j]
	})

	// Store first 3 cards for this round
	i.cards = allCards[:3]
	i.commitments = make([][32]byte, 3)
	i.salts = make([]string, 3)

	// Prepare commitment and salt for each card
	for turnIdx := 0; turnIdx < 3; turnIdx++ {
		// Generate salt for this turn
		salt := make([]byte, 32)
		crand.Read(salt)
		i.salts[turnIdx] = string(salt)

		// Calculate commitment hash for this card
		cardStr := fmt.Sprintf("%d", i.cards[turnIdx])
		hh := sha3.NewLegacyKeccak256()
		hh.Write([]byte(cardStr))
		hh.Write(salt)
		commitment := hh.Sum(nil)
		copy(i.commitments[turnIdx][:], commitment)
	}
}

type gameInfo struct {
	id           uint
	currentRound roundInfo
	maxRounds    uint32
	maxTurns     uint32
}

type Bot struct {
	ctx         context.Context
	w           *playerWallet
	mimicPlayer bool
	currentGame *gameInfo
	addr        *types.PlayerAddress
	client      *rpc.Client
	ethClient   *ethclient.Client
	bindOpt     *bind.TransactOpts
	chanEvt     chan *proto.Event
	chanErr     chan error
}

func NewBot(
	ctx context.Context,
	playerWallet *playerWallet,
	client *rpc.Client,
	ethClient *ethclient.Client,
	chainID *big.Int,
) *Bot {
	addr := types.NewPlayerAddress(playerWallet.playerId, playerWallet.tempWallet.GetAddrHex())
	opt := &bind.TransactOpts{
		From:    playerWallet.tempWallet.GetAddr(),
		Context: ctx,
		Signer:  playerWallet.tempWallet.BuildTxSinger(chainID),
	}
	return &Bot{
		ctx:       ctx,
		w:         playerWallet,
		addr:      addr,
		client:    client,
		ethClient: ethClient,
		bindOpt:   opt,
		chanEvt:   make(chan *proto.Event, 1),
		chanErr:   make(chan error, 1),
	}
}

func (b *Bot) formatBotID() string {
	return fmt.Sprintf("bot_%s", b.addr.String())
}

func (b *Bot) resubscribe(subId string, sleepTime time.Duration) error {
	err := b.client.PubSubClient.Unsubscribe(b.addr.String(), subId)
	if err != nil {
		return err
	}
	b.chanErr = make(chan error, 1)
	b.chanEvt = make(chan *proto.Event, 1)
	time.Sleep(sleepTime)
	err = b.client.PubSubClient.Subscribe(b.addr.String(), subId, b.chanEvt, b.chanErr)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) run() error {
	subId := b.formatBotID()
	err := b.client.PubSubClient.Subscribe(b.addr.String(), subId, b.chanEvt, b.chanErr)
	if err != nil {
		return err
	}
	needReconnect := false
	for {
		select {
		case <-b.ctx.Done():
			log.Infow("bot canceled", "addr", b.addr.String())
			return nil
		default:
		}
		if needReconnect {
			// make sure the old game expires
			err := b.resubscribe(subId, time.Second*90)
			if err != nil {
				log.Errorw("cannot resubscribe", "err", err)
				time.Sleep(time.Second * 10)
				continue
			}
		}
		err = b.runGameLoop()
		if err != nil {
			needReconnect = true
			continue
		}
		needReconnect = false
	}
}

func (b *Bot) runGameLoop() error {
	if b.mimicPlayer {
		if b.currentGame == nil {
			err := b.client.RpcClient.JoinQueue(b.ctx, b.addr)
			if err != nil {
				return err
			}
			log.Infow("bot start, join queue", "addr", b.addr.String())
		}
	} else {
		log.Infow("bot start, waitting for task", "addr", b.addr.String())
	}
	for {
		select {
		case <-b.ctx.Done():
			log.Infof("context done, bot exit")
			return nil
		case evt, ok := <-b.chanEvt:
			if !ok {
				break
			}
			switch evt.Type {
			case proto.EventType_TYPE_KNOWN:
				log.Errorf("unhandled event type from: %s", b.addr)
				return errors.New("bot received unexpected event: proto.EventType_TYPE_KNOWN")
			case proto.EventType_TYPE_MATCHED:
				matched := evt.GetGameMatched()
				if matched == nil {
					log.Errorw("game matched event missing GameMatched data", "addr", b.addr.String())
					continue
				}
				// Initialize game info
				b.currentGame = &gameInfo{
					id:        uint(matched.GameId),
					maxRounds: matched.MaxRoundNum,
					maxTurns:  matched.MaxTurnNum,
					currentRound: roundInfo{
						roundNum:    1,
						turnNumber:  0, // Will be set when turn ready event is received
						commitments: nil,
						cards:       nil,
						salts:       nil,
					},
				}
				err := b.client.RpcClient.ConfirmBattle(b.ctx, b.addr, uint(matched.GameId), 1, 1)
				if err != nil {
					log.Errorw("error confirm battle", "err", err, "game id", matched.GameId)
				}
			case proto.EventType_TYPE_PART_CONFIRMED:
				if b.currentGame == nil {
					log.Errorw("part confirmed but no current game", "addr", b.addr.String())
					continue
				}
				log.Infow("player part confirmed", "game id", b.currentGame.id)
			case proto.EventType_TYPE_GAME_CREATED:
				if b.currentGame == nil {
					log.Errorw("game created but no current game", "addr", b.addr.String())
					continue
				}
				log.Infow("game created", "game id", b.currentGame.id)
			case proto.EventType_TYPE_ROUND_READY:
				if b.currentGame == nil {
					log.Errorw("round ready but no current game", "addr", b.addr.String())
					continue
				}
				roundReady := evt.GetRoundReady()
				if roundReady == nil {
					log.Errorw("round ready event missing RoundReady data", "addr", b.addr.String())
					continue
				}
				// Validate round number
				expectedRoundNum := b.currentGame.currentRound.roundNum
				if uint(roundReady.RoundNum) != expectedRoundNum {
					log.Errorw("round number mismatch", "expected", expectedRoundNum, "received", roundReady.RoundNum, "game id", b.currentGame.id)
					continue
				}
				log.Infow("round ready", "game id", b.currentGame.id, "round", roundReady.RoundNum)
				// Prepare 3 cards, salts, and commitments for this round
				b.currentGame.currentRound.roundNum = uint(roundReady.RoundNum)
				b.currentGame.currentRound.turnNumber = 0 // Reset turn number
				b.currentGame.currentRound.prepareCards()
			case proto.EventType_TYPE_TURN_READY:
				if b.currentGame == nil {
					log.Errorw("turn ready but no current game", "addr", b.addr.String())
					continue
				}
				turnReady := evt.GetTurnReady()
				if turnReady == nil {
					log.Errorw("turn ready event missing TurnReady data", "addr", b.addr.String())
					continue
				}
				// Validate round and turn numbers
				expectedRoundNum := b.currentGame.currentRound.roundNum
				expectedTurnNum := b.currentGame.currentRound.turnNumber + 1 // Next expected turn (1-3)
				if uint(turnReady.RoundNum) != expectedRoundNum {
					log.Errorw("round number mismatch in turn ready", "expected", expectedRoundNum, "received", turnReady.RoundNum, "game id", b.currentGame.id)
					continue
				}
				if turnReady.TurnNum < 1 || turnReady.TurnNum > 3 {
					log.Errorw("invalid turn number", "turn", turnReady.TurnNum, "game id", b.currentGame.id)
					continue
				}
				if turnReady.TurnNum != expectedTurnNum {
					log.Errorw("turn number mismatch in turn ready", "expected", expectedTurnNum, "received", turnReady.TurnNum, "current turn", b.currentGame.currentRound.turnNumber, "game id", b.currentGame.id)
					continue
				}
				log.Infow("turn ready", "game id", b.currentGame.id, "round", turnReady.RoundNum, "turn", turnReady.TurnNum)
				// Update turn number
				b.currentGame.currentRound.turnNumber = turnReady.TurnNum
				// Submit commitment for this turn using turn number - 1 as index
				turnIdx := int(turnReady.TurnNum) - 1
				if turnIdx >= 0 && turnIdx < len(b.currentGame.currentRound.commitments) {
					err := b.client.RpcClient.SubmitPlayerCommitment(
						b.ctx,
						b.addr,
						turnReady.RoundNum,
						b.currentGame.currentRound.commitments[turnIdx][:],
						turnReady.TurnNum,
						nil, // Signature - empty for bots
						b.currentGame.id,
					)
					if err != nil {
						log.Errorw("submit commitment failed", "err", err, "game id", b.currentGame.id, "round", turnReady.RoundNum, "turn", turnReady.TurnNum)
					} else {
						log.Infow("submitted commitment", "game id", b.currentGame.id, "round", turnReady.RoundNum, "turn", turnReady.TurnNum)
					}
				} else {
					log.Errorw("invalid turn index for commitment", "turn", turnReady.TurnNum, "game id", b.currentGame.id)
				}
			case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
				if b.currentGame == nil {
					log.Errorw("commitments on chain but no current game", "addr", b.addr.String())
					continue
				}
				commitmentsOnChain := evt.GetCommitmentsOnChain()
				if commitmentsOnChain == nil {
					log.Errorw("commitments on chain event missing CommitmentsOnChain data", "addr", b.addr.String())
					continue
				}
				// Validate round and turn numbers
				expectedRoundNum := b.currentGame.currentRound.roundNum
				expectedTurnNum := b.currentGame.currentRound.turnNumber
				if uint(commitmentsOnChain.RoundNum) != expectedRoundNum {
					log.Errorw("round number mismatch in commitments on chain", "expected", expectedRoundNum, "received", commitmentsOnChain.RoundNum, "game id", b.currentGame.id)
					continue
				}
				if commitmentsOnChain.TurnNum != expectedTurnNum {
					log.Errorw("turn number mismatch in commitments on chain", "expected", expectedTurnNum, "received", commitmentsOnChain.TurnNum, "game id", b.currentGame.id)
					continue
				}
				log.Infow("commitments on chain", "game id", b.currentGame.id, "round", commitmentsOnChain.RoundNum, "turn", commitmentsOnChain.TurnNum)
				// Submit card and salt for the current turn using turn number - 1 as index
				turnNumber := commitmentsOnChain.TurnNum
				turnIdx := int(turnNumber) - 1
				if turnIdx < 0 || turnIdx >= 3 {
					log.Errorw("invalid turn number", "turn", turnNumber, "game id", b.currentGame.id)
					continue
				}
				cardID := b.currentGame.currentRound.cards[turnIdx]
				salt := b.currentGame.currentRound.salts[turnIdx]
				err := b.client.RpcClient.SubmitPlayerCard(
					b.ctx,
					b.addr,
					commitmentsOnChain.RoundNum,
					[]byte(salt),
					uint(cardID),
					turnNumber,
					nil, // Signature - empty for bots
					b.currentGame.id,
				)
				if err != nil {
					log.Errorw("submit card failed", "err", err, "game id", b.currentGame.id, "round", commitmentsOnChain.RoundNum, "turn", turnNumber)
				} else {
					log.Infow("submitted card", "game id", b.currentGame.id, "round", commitmentsOnChain.RoundNum, "turn", turnNumber, "card", cardID)
				}
			case proto.EventType_TYPE_TURN_COMPLETE:
				if b.currentGame == nil {
					log.Errorw("turn complete but no current game", "addr", b.addr.String())
					continue
				}
				turnCompleted := evt.GetTurnCompleted()
				if turnCompleted == nil {
					log.Errorw("turn complete event missing TurnCompleted data", "addr", b.addr.String())
					continue
				}
				// Validate round and turn numbers
				expectedRoundNum := b.currentGame.currentRound.roundNum
				expectedTurnNum := b.currentGame.currentRound.turnNumber
				if uint(turnCompleted.RoundNum) != expectedRoundNum {
					log.Errorw("round number mismatch in turn complete", "expected", expectedRoundNum, "received", turnCompleted.RoundNum, "game id", b.currentGame.id)
					continue
				}
				if turnCompleted.TurnNum != expectedTurnNum {
					log.Errorw("turn number mismatch in turn complete", "expected", expectedTurnNum, "received", turnCompleted.TurnNum, "game id", b.currentGame.id)
					continue
				}
				log.Infow("turn complete", "game id", b.currentGame.id, "round", turnCompleted.RoundNum, "turn", turnCompleted.TurnNum, "isRoundComplete", turnCompleted.IsRoundComplete, "isGameComplete", turnCompleted.IsGameComplete)

				// Handle game completion
				if turnCompleted.IsGameComplete {
					log.Infow("game complete", "game id", b.currentGame.id)
					// Log game result if available
					if turnCompleted.GameResult != nil {
						log.Infow("game result", "game id", b.currentGame.id, "winner", turnCompleted.GameResult.WinnerPlayerId, "result type", turnCompleted.GameResult.GameResultType)
					}
					// Send continue canceled and wait for another game
					err := b.client.RpcClient.RefuseContinueGame(b.ctx, b.addr, b.currentGame.id)
					if err != nil {
						log.Errorw("error refuse continue game", "err", err)
					}
					b.currentGame = nil
					return nil
				}

				// Handle round completion
				if turnCompleted.IsRoundComplete {
					log.Infow("round complete", "game id", b.currentGame.id, "round", turnCompleted.RoundNum)
					// Round number increases by 1 and prepare for next round
					b.currentGame.currentRound.prepareNewRound()
					// Prepare cards for the next round
					b.currentGame.currentRound.prepareCards()
					// Confirm battle for the next round (turn 1)
					err := b.client.RpcClient.ConfirmBattle(b.ctx, b.addr, b.currentGame.id, b.currentGame.currentRound.roundNum, 1)
					if err != nil {
						log.Errorw("confirm battle failed", "err", err, "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
					} else {
						log.Infow("confirmed battle for next round", "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
					}
				} else {
					// Otherwise, just prepare for next turn (no action needed, will wait for next turn ready event)
					log.Debugw("turn complete, waiting for next turn", "game id", b.currentGame.id, "round", turnCompleted.RoundNum, "turn", turnCompleted.TurnNum)
				}
			}
		case err, ok := <-b.chanErr:
			if !ok {
				break
			}
			return err
		}
	}
}
