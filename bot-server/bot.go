package botserver

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand/v2"

	gameclient "github.com/CryptoElementals/common/game_client"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/utils"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
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
	commitments [][]byte
	cards       []uint32 // Store cards as array for easier access
	salts       []string
}

// GetCard makes roundInfo implement gameclient.CardProvider.
// It selects a card for the given round/turn using its internally prepared cards.
func (i *roundInfo) GetCard(round uint32, turn uint32) (uint32, error) {
	// If this is a new round or cards are not prepared, prepare them.
	if i.roundNum != uint(round) || len(i.cards) == 0 {
		i.roundNum = uint(round)
		i.turnNumber = 1
		i.cards = nil
		i.commitments = nil
		i.salts = nil
		i.prepareCards()
	}

	if turn == 0 {
		return 0, fmt.Errorf("invalid turn number: %d", turn)
	}
	idx := int(turn - 1)
	if idx < 0 || idx >= len(i.cards) {
		return 0, fmt.Errorf("turn index out of range: %d (len=%d)", idx, len(i.cards))
	}

	return i.cards[idx], nil
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
	i.commitments = make([][]byte, 3)
	i.salts = make([]string, 3)

	// Prepare commitment and salt for each card
	for turnIdx := range 3 {
		// Generate salt for this turn
		salt := make([]byte, 32)
		crand.Read(salt)
		i.salts[turnIdx] = string(salt)

		// Calculate commitment hash for this card using SolidityPackedKeccak256
		hash, _ := utils.SolidityPackedKeccak256(
			[]any{
				i.cards[turnIdx],
				salt,
			},
		)
		i.commitments[turnIdx] = hash.Bytes()
	}
}

type Bot struct {
	ctx          context.Context
	w            *playerWallet
	addr         *types.PlayerAddress
	client       *rpc.Client
	ethClient    *ethclient.Client
	bindOpt      *bind.TransactOpts
	chanEvt      chan *proto.Event
	chanErr      chan error
	cardProvider gameclient.CardProvider
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
		cardProvider: &roundInfo{
			roundNum:   1,
			turnNumber: 1,
		},
	}
}

func (b *Bot) formatBotID() string {
	return fmt.Sprintf("bot_%s", b.addr.String())
}

func (b *Bot) run() error {
	err := b.runGameContext()
	if err != nil {
		return err
	}
	return nil
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
