package botserver

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	gameclient "github.com/CryptoElementals/common/game_client"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type playerWallet struct {
	playerId   int64
	tempWallet *wallet.Wallet
}

func (w *playerWallet) address() *types.PlayerAddress {
	return types.NewPlayerAddress(w.playerId, w.tempWallet.GetAddrHex())
}

type roundInfo struct {
	wallet         *wallet.Wallet
	roundNum       uint
	cards          []uint32
	fixedOpponents map[int64]struct{}
	fixedCardOrder []uint32
}

func newRoundInfo(w *wallet.Wallet, fixedOpponentIDs []int64, fixedCards []uint32) *roundInfo {
	fo := make(map[int64]struct{}, len(fixedOpponentIDs))
	for _, id := range fixedOpponentIDs {
		fo[id] = struct{}{}
	}
	seq := fixedCards
	if len(seq) == 0 {
		seq = []uint32{2, 3, 5}
	}
	return &roundInfo{
		wallet:         w,
		roundNum:       1,
		fixedOpponents: fo,
		fixedCardOrder: seq,
	}
}

// GetCard makes roundInfo implement gameclient.CardProvider.
func (i *roundInfo) GetCard(ctx gameclient.CardPickContext) (uint32, error) {
	if ctx.Turn == 0 {
		return 0, fmt.Errorf("invalid turn number: %d", ctx.Turn)
	}
	if ctx.GameID == 0 {
		return 0, fmt.Errorf("game id not set")
	}

	if i.roundNum != uint(ctx.Round) || len(i.cards) == 0 {
		i.roundNum = uint(ctx.Round)
		if err := i.prepareRound(ctx); err != nil {
			return 0, err
		}
	}

	idx := int(ctx.Turn - 1)
	if idx < 0 || idx >= len(i.cards) {
		return 0, fmt.Errorf("turn index out of range: %d (len=%d)", idx, len(i.cards))
	}

	return i.cards[idx], nil
}

func (i *roundInfo) prepareRound(ctx gameclient.CardPickContext) error {
	if _, useFixed := i.fixedOpponents[ctx.OpponentID]; useFixed {
		log.Debugw("found whitelisted opponent player", "id", ctx.OpponentID)
		i.cards = append([]uint32(nil), i.fixedCardOrder...)
		return nil
	}
	cards, err := gameclient.DeriveRoundThreeCards(i.wallet, ctx.GameID, ctx.Round)
	if err != nil {
		return err
	}
	i.cards = cards
	return nil
}

type Bot struct {
	ctx          context.Context
	w            *playerWallet
	addr         *types.PlayerAddress
	client       *rpc.Client
	bindOpt      *bind.TransactOpts
	cardProvider gameclient.CardProvider
	mode         string
	apiBaseURL   string
}

func NewBot(
	ctx context.Context,
	playerWallet *playerWallet,
	client *rpc.Client,
	chainID *big.Int,
	mode string,
	apiBaseURL string,
	fixedOpponentPlayerIDs []int64,
	fixedCardSequence []uint32,
) *Bot {
	addr := types.NewPlayerAddress(playerWallet.playerId, playerWallet.tempWallet.GetAddrHex())
	opt := &bind.TransactOpts{
		From:    playerWallet.tempWallet.GetAddr(),
		Context: ctx,
		Signer:  playerWallet.tempWallet.BuildTxSinger(chainID),
	}
	return &Bot{
		ctx:     ctx,
		w:       playerWallet,
		addr:    addr,
		client:  client,
		bindOpt: opt,
		cardProvider: newRoundInfo(
			playerWallet.tempWallet,
			fixedOpponentPlayerIDs,
			fixedCardSequence,
		),
		mode:       mode,
		apiBaseURL: apiBaseURL,
	}
}

func (b *Bot) formatBotID() string {
	return fmt.Sprintf("bot_%s", b.addr.String())
}

func (b *Bot) run() error {
	backoff := 500 * time.Millisecond
	const maxBackoff = 5 * time.Second
	for {
		var err error
		if b.mode == "http" {
			err = b.runGameContextHTTP()
		} else {
			err = b.runGameContext()
		}
		if err == nil {
			if b.ctx.Err() != nil {
				return nil
			}
			backoff = 500 * time.Millisecond
			continue
		}
		if b.ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return nil
		}
		log.Errorw("bot_stream_subscribe_failed", "err", err, "addr", b.addr.String(), "retry_in", backoff.String())
		select {
		case <-b.ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
		log.Infow("bot_stream_subscribe_retrying", "addr", b.addr.String())
	}
}

func (b *Bot) runGameContextHTTP() error {
	gameContext, err := gameclient.NewGameContextHTTP(b.ctx, b.apiBaseURL, b.w.playerId, b.w.tempWallet, b.cardProvider)
	if err != nil {
		return err
	}
	if err := gameContext.SignIn(); err != nil {
		return err
	}
	if err := gameContext.Subscribe(); err != nil {
		return err
	}
	defer gameContext.Close()
	for {
		select {
		case <-b.ctx.Done():
			return nil
		default:
			err = gameContext.Run()
			if err != nil {
				return err
			}
		}
	}
}

func (b *Bot) runGameContext() error {
	gameContext, err := gameclient.NewGameContext(b.ctx, b.w.playerId, b.w.tempWallet, b.client, b.cardProvider)
	if err != nil {
		return err
	}
	err = gameContext.Subscribe(b.formatBotID())
	if err != nil {
		return err
	}
	if err := gameContext.SyncGamePhaseIfInGame(); err != nil {
		return err
	}
	for {
		select {
		case <-b.ctx.Done():
			return nil
		default:
			err = gameContext.Run()
			if err != nil {
				return err
			}
		}
	}
}
