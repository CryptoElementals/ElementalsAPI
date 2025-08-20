package botserver

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"time"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

type PlayerWallet struct {
	accountWallet *wallet.Wallet
	tempWallet    *wallet.Wallet
}

func (w *PlayerWallet) Address() *types.PlayerAddress {
	return types.NewPlayerAddress(w.accountWallet.GetAddrHex(), w.tempWallet.GetAddrHex())
}

type roundInfo struct {
	roundNum   uint
	commitment [32]byte
	cards      string
	salt       string
}

func (i *roundInfo) prepareNewRound() {
	i.roundNum++
	i.cards = ""
	i.commitment = [32]byte{}
	i.salt = ""
}

func (i *roundInfo) prepareCards() {
	// select random cards
	cards := make([]uint32, 5)
	for i := range cards {
		cards[i] = uint32(i + 1)
	}
	rand.Shuffle(5, func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})

	// calculate commitment
	cardsStr := fmt.Sprintf("%d,%d,%d", cards[0], cards[1], cards[2])
	i.cards = cardsStr
	salt := make([]byte, 32)
	crand.Read(salt)
	i.salt = string(salt)
	// 计算承诺哈希
	hh := sha3.NewLegacyKeccak256()
	hh.Write([]byte(cardsStr))
	hh.Write(salt)
	commitment := hh.Sum(nil)
	i.commitment = [32]byte(commitment)
}

type gameInfo struct {
	id                  uint
	currentRound        roundInfo
	gameContractAddress string
	gameContract        *contract.RoomContract
}

type Bot struct {
	ctx         context.Context
	w           *PlayerWallet
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
	playerWallet *PlayerWallet,
	client *rpc.Client,
	ethClient *ethclient.Client,
	chainID *big.Int,
) *Bot {
	addr := types.NewPlayerAddress(playerWallet.accountWallet.GetAddrHex(), playerWallet.tempWallet.GetAddrHex())
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
	err = b.recoverGameInfo()
	if err != nil {
		log.Errorw("cannot recover game", "err", err)
		needReconnect = true
	}
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

func (b *Bot) recoverGameInfo() error {
	phase, err := b.client.RpcClient.GetGamePhase(b.ctx, b.addr)
	if err != nil {
		log.Errorw("error get game phase", "err", err)
		return err
	}
	if phase.PvPInfo.Status != proto.PlayerStatus_PLAYER_IN_GAME {
		return nil
	}
	b.currentGame = &gameInfo{
		id: uint(phase.PvPInfo.GameID),
		currentRound: roundInfo{
			roundNum: uint(phase.PvPInfo.RoundNumber),
		},
	}
	log.Infow("recover game", "addr", types.ToJsonLoggable(b.addr), "game id", b.currentGame.id, "round", b.currentGame.currentRound)
	if phase.PvPInfo.ContractAddress != "" {
		c, err := contract.NewRoomContract(common.HexToAddress(phase.PvPInfo.ContractAddress), b.ethClient)
		if err != nil {
			log.Errorw("new room contract failed", "err", err, "addr", types.ToJsonLoggable(b.addr), "game id", b.currentGame.id, "round", b.currentGame.currentRound, "contract", phase.PvPInfo.ContractAddress)
			return err
		}
		b.currentGame.gameContract = c
		b.currentGame.gameContractAddress = phase.PvPInfo.ContractAddress
	}
	for _, player := range phase.Players {
		// found myself
		if player.Address.TemporaryAddress == b.addr.TemporaryAddress &&
			player.Address.WalletAddress == b.addr.WalletAddress {
			// need confirm
			if !player.IsConfirmed {
				log.Infow("recover game, confirm battle", "addr", types.ToJsonLoggable(b.addr), "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
				err := b.client.RpcClient.ConfirmBattle(b.ctx, b.addr, b.currentGame.id, b.currentGame.currentRound.roundNum)
				if err != nil {
					log.Errorw("confirm battle failed", "addr", types.ToJsonLoggable(b.addr), "err", err, "game id", b.currentGame.id, "round", b.currentGame.currentRound)
					return err
				}
				return nil
			}
			// didn't send cards
			if len(player.Commitment) == 0 {
				log.Infow("recover game, submit cards", "addr", types.ToJsonLoggable(b.addr), "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
				b.currentGame.currentRound.prepareCards()
				tx, err := b.currentGame.gameContract.SubmitCardsHash(b.bindOpt, b.currentGame.currentRound.commitment, big.NewInt(int64(b.currentGame.currentRound.roundNum)))
				if err != nil {
					log.Errorw("submit card hash failed", "addr", types.ToJsonLoggable(b.addr), "err", err, "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum, "contract", b.currentGame.gameContractAddress)
					return err
				}
				log.Infow("submitted card hash", "addr", types.ToJsonLoggable(b.addr), "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum,
					"contract", b.currentGame.gameContractAddress, "hash", hexutil.Encode(b.currentGame.currentRound.commitment[:]), "txHash", tx.Hash().String())
				return nil
			}
			return fmt.Errorf("game not recoverable, addr: %s, game: %d, round: %d", types.ToJsonLoggable(b.addr), b.currentGame.id, b.currentGame.currentRound.roundNum)
		}
	}
	return fmt.Errorf("cannot find myself from game player list, addr: %s, game: %d, round: %d", types.ToJsonLoggable(b.addr), b.currentGame.id, b.currentGame.currentRound.roundNum)
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
				}
				b.currentGame = &gameInfo{
					id: uint(phase.PvPInfo.GameID),
					currentRound: roundInfo{
						roundNum: 1,
					},
				}
				err = b.client.RpcClient.ConfirmBattle(b.ctx, b.addr, b.currentGame.id, b.currentGame.currentRound.roundNum)
				if err != nil {
					log.Errorw("error confirm battle", "err", err, "game id", b.currentGame.id)
				}
			case proto.EventType_TYPE_PART_CONFIRMED:
				log.Infow("player part confirmed", "game id", b.currentGame.id, "round", b.currentGame.currentRound)
			case proto.EventType_TYPE_GAME_CREATED:
				log.Infow("game created", "game id", b.currentGame.id)
				// get contract
				phase, err := b.client.RpcClient.GetGamePhase(b.ctx, b.addr)
				if err != nil {
					log.Errorw("get game phase failed", "err", err, "game id", b.currentGame.id)
				}

				c, err := contract.NewRoomContract(common.HexToAddress(phase.PvPInfo.ContractAddress), b.ethClient)
				if err != nil {
					log.Errorw("new room contract failed", "err", err, "game id", b.currentGame.id, "round", b.currentGame.currentRound, "contract", phase.PvPInfo.ContractAddress)

				}
				b.currentGame.gameContractAddress = phase.PvPInfo.ContractAddress
				b.currentGame.gameContract = c
			case proto.EventType_TYPE_ROUND_READY:
				log.Infow("round ready", "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
				// submit commitments
				b.currentGame.currentRound.prepareCards()
				tx, err := b.currentGame.gameContract.SubmitCardsHash(b.bindOpt, b.currentGame.currentRound.commitment, big.NewInt(int64(b.currentGame.currentRound.roundNum)))
				if err != nil {
					log.Errorw("submit card hash failed", "err", err, "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum, "contract", b.currentGame.gameContractAddress)
				}
				log.Infow("submitted card hash", "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum,
					"contract", b.currentGame.gameContractAddress, "hash", hexutil.Encode(b.currentGame.currentRound.commitment[:]), "txHash", tx.Hash().String())
			case proto.EventType_TYPE_COMMITMENTS_ON_CHAIN:
				log.Infow("commitments on chain", "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
				// submit cards
				tx, err := b.currentGame.gameContract.SubmitCards(b.bindOpt, b.currentGame.currentRound.cards, b.currentGame.currentRound.salt, big.NewInt(int64(b.currentGame.currentRound.roundNum)))
				if err != nil {
					log.Errorw("submit card hash failed", "err", err, "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum, "contract", b.currentGame.gameContractAddress)
				}
				log.Infow("submitted cards", "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum,
					"contract", b.currentGame.gameContractAddress, "cards", b.currentGame.currentRound.cards, "txHash", tx.Hash().String())
			case proto.EventType_TYPE_CARDS_ON_CHAIN:
				log.Infow("cards on chain", "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
			case proto.EventType_TYPE_ROUND_COMPLETE:
				log.Infow("round complete", "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
				battleInfo, err := b.client.RpcClient.GetBattleInfo(b.ctx, b.currentGame.id, b.currentGame.currentRound.roundNum)
				if err != nil {
					log.Errorw("get battle info failed", "err", err, "game id", b.currentGame.id, "round", b.currentGame.currentRound.roundNum)
					continue
				}
				if !battleInfo.RoundResult.IsGameOver {
					b.currentGame.currentRound.prepareNewRound()
					b.client.RpcClient.ConfirmBattle(b.ctx, b.addr, b.currentGame.id, b.currentGame.currentRound.roundNum)
					log.Infof("confirm submitted, addr: %s, round %d, game: %d", b.addr.String(), b.currentGame.currentRound.roundNum, b.currentGame.id)
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
