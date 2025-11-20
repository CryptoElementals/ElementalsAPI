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
	roundNum   uint
	commitment [32]byte
	cards      []uint32 // Store cards as array for easier access
	salt       string
	turnNumber uint32 // Track current turn number (1, 2, or 3)
}

func (i *roundInfo) prepareNewRound() {
	i.roundNum++
	i.cards = nil
	i.commitment = [32]byte{}
	i.salt = ""
	i.turnNumber = 0
}

func (i *roundInfo) prepareCards() {
	// select random cards
	cards := make([]uint32, 5)
	for j := range cards {
		cards[j] = uint32(j + 1)
	}
	rand.Shuffle(5, func(j, k int) {
		cards[j], cards[k] = cards[k], cards[j]
	})

	// Store first 3 cards for this round
	i.cards = cards[:3]
	salt := make([]byte, 32)
	crand.Read(salt)
	i.salt = string(salt)
	// Calculate commitment hash
	cardsStr := fmt.Sprintf("%d,%d,%d", i.cards[0], i.cards[1], i.cards[2])
	hh := sha3.NewLegacyKeccak256()
	hh.Write([]byte(cardsStr))
	hh.Write(salt)
	commitment := hh.Sum(nil)
	i.commitment = [32]byte(commitment)
}

type gameInfo struct {
	id           uint
	currentRound roundInfo
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
				log.Infow("bot matched")
				phase, err := b.client.RpcClient.GetGamePhase(b.ctx, b.addr)
				if err != nil {
					log.Errorw("error get game phase", "err", err)
					continue
				}
				b.currentGame = &gameInfo{
					id: uint(phase.GameID),
					currentRound: roundInfo{
						roundNum:   uint(phase.RoundNumber),
						turnNumber: phase.TurnNumber,
					},
				}
				err = b.client.RpcClient.ConfirmBattle(b.ctx, b.addr, b.currentGame.id, b.currentGame.currentRound.roundNum, b.currentGame.currentRound.turnNumber)
				if err != nil {
					log.Errorw("error confirm battle", "err", err, "game id", b.currentGame.id)
				}
			case proto.EventType_TYPE_PART_CONFIRMED:
				log.Infow("player part confirmed", "game id", b.currentGame.id, "round", b.currentGame.currentRound)
			case proto.EventType_TYPE_GAME_CREATED:
				log.Infow("game created", "game id", b.currentGame.id)
				// Game created, prepare for first round
				b.currentGame.currentRound.roundNum = 1
				b.currentGame.currentRound.turnNumber = 0
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
				log.Infow("round ready", "game id", b.currentGame.id, "round", roundReady.RoundNum)
				// Prepare cards for this round
				b.currentGame.currentRound.roundNum = uint(roundReady.RoundNum)
				b.currentGame.currentRound.prepareCards()
				// Submit commitment for turn 1
				err := b.client.RpcClient.SubmitPlayerCommitment(
					b.ctx,
					b.addr,
					roundReady.RoundNum,
					b.currentGame.currentRound.commitment[:],
					1,   // Turn number 1
					nil, // Signature - empty for bots
					b.currentGame.id,
				)
				if err != nil {
					log.Errorw("submit commitment failed", "err", err, "game id", b.currentGame.id, "round", roundReady.RoundNum)
				} else {
					log.Infow("submitted commitment", "game id", b.currentGame.id, "round", roundReady.RoundNum, "turn", 1)
				}
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
				log.Infow("turn ready", "game id", b.currentGame.id, "round", turnReady.RoundNum, "turn", turnReady.TurnNum)
				// If this is the first turn of a round and we haven't prepared cards yet, prepare them
				if turnReady.TurnNum == 1 && len(b.currentGame.currentRound.cards) == 0 {
					b.currentGame.currentRound.roundNum = uint(turnReady.RoundNum)
					b.currentGame.currentRound.prepareCards()
				}
				// Submit commitment for this turn
				if int(turnReady.TurnNum) <= len(b.currentGame.currentRound.cards) {
					err := b.client.RpcClient.SubmitPlayerCommitment(
						b.ctx,
						b.addr,
						turnReady.RoundNum,
						b.currentGame.currentRound.commitment[:],
						turnReady.TurnNum,
						nil, // Signature - empty for bots
						b.currentGame.id,
					)
					if err != nil {
						log.Errorw("submit commitment failed", "err", err, "game id", b.currentGame.id, "round", turnReady.RoundNum, "turn", turnReady.TurnNum)
					} else {
						log.Infow("submitted commitment", "game id", b.currentGame.id, "round", turnReady.RoundNum, "turn", turnReady.TurnNum)
					}
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
				log.Infow("commitments on chain", "game id", b.currentGame.id, "round", commitmentsOnChain.RoundNum, "turn", commitmentsOnChain.CardNum)
				// Submit card for the current turn
				turnNumber := commitmentsOnChain.CardNum
				if turnNumber < 1 || turnNumber > 3 {
					log.Errorw("invalid turn number", "turn", turnNumber, "game id", b.currentGame.id)
					continue
				}
				if len(b.currentGame.currentRound.cards) < int(turnNumber) {
					log.Errorw("no card prepared for turn", "turn", turnNumber, "game id", b.currentGame.id)
					continue
				}
				cardID := b.currentGame.currentRound.cards[turnNumber-1]
				err := b.client.RpcClient.SubmitPlayerCard(
					b.ctx,
					b.addr,
					commitmentsOnChain.RoundNum,
					[]byte(b.currentGame.currentRound.salt),
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
			case proto.EventType_TYPE_CARDS_ON_CHAIN:
				if b.currentGame == nil {
					log.Errorw("cards on chain but no current game", "addr", b.addr.String())
					continue
				}
				cardsOnChain := evt.GetCardsOnChain()
				if cardsOnChain == nil {
					log.Errorw("cards on chain event missing CardsOnChain data", "addr", b.addr.String())
					continue
				}
				log.Infow("cards on chain", "game id", b.currentGame.id, "round", cardsOnChain.RoundNum)
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
				log.Infow("turn complete", "game id", b.currentGame.id, "round", turnCompleted.RoundNum, "turn", turnCompleted.TurnNum)
				// Update turn number for next turn
				b.currentGame.currentRound.turnNumber = turnCompleted.TurnNum
			case proto.EventType_TYPE_ROUND_COMPLETE:
				if b.currentGame == nil {
					log.Errorw("round complete but no current game", "addr", b.addr.String())
					continue
				}
				roundCompleted := evt.GetRoundCompleted()
				if roundCompleted == nil {
					log.Errorw("round complete event missing RoundCompleted data", "addr", b.addr.String())
					continue
				}
				log.Infow("round complete", "game id", b.currentGame.id, "round", roundCompleted.RoundNumber)
				// Get battle info to check if game is over
				battleInfo, err := b.client.RpcClient.GetBattleInfo(b.ctx, b.currentGame.id, uint(roundCompleted.RoundNumber))
				if err != nil {
					log.Errorw("get battle info failed", "err", err, "game id", b.currentGame.id, "round", roundCompleted.RoundNumber)
					continue
				}
				if !battleInfo.RoundResult.IsGameOver {
					// Game continues, prepare for next round
					b.currentGame.currentRound.prepareNewRound()
					err = b.client.RpcClient.ConfirmBattle(b.ctx, b.addr, b.currentGame.id, b.currentGame.currentRound.roundNum, b.currentGame.currentRound.turnNumber)
					if err != nil {
						log.Errorw("confirm battle failed", "err", err, "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
					} else {
						log.Infow("confirmed battle for next round", "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
					}
				} else {
					log.Infow("game over", "game id", b.currentGame.id)
				}
			case proto.EventType_TYPE_GAME_COMPLETE:
				log.Infow("game complete", "game id", b.currentGame.id)
				err := b.client.RpcClient.RefuseContinueGame(b.ctx, b.addr, b.currentGame.id)
				if err != nil {
					log.Errorw("error refuse continue game", "err", err)
				}
				b.currentGame = nil
				// skip continue

				return nil
			}
		case err, ok := <-b.chanErr:
			if !ok {
				break
			}
			return err
		}
	}
}
