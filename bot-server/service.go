package botserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	gameclient "github.com/CryptoElementals/common/game_client"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	ctx               context.Context
	ccl               context.CancelFunc
	bots              []*Bot
	addresses         []*types.PlayerAddress
	rpcClient         *rpc.Client
	wg                sync.WaitGroup
	mu                sync.Mutex
	started           bool
	registered        bool
	mode              string
	registry          *redisBotRegistry
	heartbeatInterval time.Duration
	heartbeatCancel   context.CancelFunc
	heartbeatWG       sync.WaitGroup
}

func parseWallet(path config.WalletInfo) (*playerWallet, error) {
	tempWallet, err := wallet.LoadWallet(path.TemporaryWallet)
	if err != nil {
		return nil, err
	}

	return &playerWallet{
		playerId:   path.PlayerId,
		tempWallet: tempWallet,
	}, nil
}

func NewService(
	ctx context.Context,
	cfg *config.BotConfig,
) (*Service, error) {
	chainClient, err := ethclient.Dial(cfg.ChainCfg.HttpRpc)
	if err != nil {
		return nil, err
	}
	chainID, err := chainClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	rpcClient, err := rpc.NewClient(cfg.RoomServerEndpoint, cfg.LobbyServerEndpoint)
	if err != nil {
		return nil, err
	}
	playerWallets := make([]*playerWallet, 0)
	mode := strings.ToLower(strings.TrimSpace(cfg.GameClientMode))
	if mode == "" {
		mode = "grpc"
	}
	if cfg.NumBots > 0 {
		playerWallets, err = ensureBotAccounts(ctx, cfg.NumBots, cfg.ApiServerEndpoint)
		if err != nil {
			return nil, err
		}
	} else {
		playerWallets = make([]*playerWallet, 0, len(cfg.WalletInfos))
		for _, walletInfo := range cfg.WalletInfos {
			p, parseErr := parseWallet(walletInfo)
			if parseErr != nil {
				return nil, parseErr
			}
			playerWallets = append(playerWallets, p)
		}
	}

	bots := make([]*Bot, 0, len(playerWallets))
	addresses := make([]*types.PlayerAddress, 0, len(playerWallets))
	ctx, ccl := context.WithCancel(ctx)
	log.Debugw("set fixed card order players", "ids", cfg.FixedCardOpponentPlayerIds, "cards", cfg.FixedCardSequence)
	for _, p := range playerWallets {
		b := NewBot(ctx, p, rpcClient, chainID, mode, cfg.ApiServerEndpoint, cfg.FixedCardOpponentPlayerIds, cfg.FixedCardSequence)
		bots = append(bots, b)
		addresses = append(addresses, p.address())
	}
	var registry *redisBotRegistry
	if r, regErr := newRedisBotRegistry(""); regErr != nil {
		log.Warnw("bot registry redis not available", "err", regErr)
	} else {
		registry = r
	}
	return &Service{
		ctx:               ctx,
		ccl:               ccl,
		rpcClient:         rpcClient,
		bots:              bots,
		addresses:         addresses,
		mode:              mode,
		registry:          registry,
		heartbeatInterval: time.Duration(cfg.BotRegistryHeartbeatIntervalSec) * time.Second,
	}, nil
}

func (s *Service) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	if s.registry == nil {
		s.mu.Unlock()
		return fmt.Errorf("redis bot registry is required but not available")
	}
	if err := s.upsertBotsAliveNow(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("upsert bots alive to redis: %w", err)
	}
	s.startHeartbeatLoopLocked()
	s.registered = true
	s.started = true
	s.mu.Unlock()

	log.Infow("bot_register_success", "count", len(s.addresses))
	s.runBots()
	return nil
}

func ensureBotAccounts(ctx context.Context, numBots int, apiServerEndpoint string) ([]*playerWallet, error) {
	if numBots <= 0 {
		return nil, fmt.Errorf("num bots must be positive")
	}
	existing, err := db.ListBotAccounts(numBots)
	if err != nil {
		return nil, fmt.Errorf("list bot accounts: %w", err)
	}
	if len(existing) < numBots {
		if apiServerEndpoint == "" {
			return nil, fmt.Errorf("api-server-endpoint is required to provision missing bot accounts")
		}
		missing := numBots - len(existing)
		for i := 0; i < missing; i++ {
			account, createErr := provisionBotAccountByHTTP(ctx, apiServerEndpoint)
			if createErr != nil {
				return nil, fmt.Errorf("provision bot account: %w", createErr)
			}
			existing = append(existing, *account)
		}
	}
	out := make([]*playerWallet, 0, numBots)
	for _, acc := range existing[:numBots] {
		w, walletErr := wallet.NewWalletFromPrivKeyHex(acc.TempPrivateKey)
		if walletErr != nil {
			return nil, fmt.Errorf("build wallet for player %d: %w", acc.PlayerID, walletErr)
		}
		out = append(out, &playerWallet{
			playerId:   acc.PlayerID,
			tempWallet: w,
		})
	}
	return out, nil
}

func provisionBotAccountByHTTP(ctx context.Context, apiServerEndpoint string) (*dao.BotAccount, error) {
	w, err := wallet.NewWallet("")
	if err != nil {
		return nil, fmt.Errorf("create temp wallet: %w", err)
	}
	addr := strings.ToLower(w.GetAddrHex())
	apiClient, err := gameclient.NewHttpApiClient(ctx, apiServerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("new http api client: %w", err)
	}
	nonce, loginCode, err := apiClient.GetLoginCode(addr)
	if err != nil {
		return nil, fmt.Errorf("get login code: %w", err)
	}
	signature, err := w.EthSign(loginCode)
	if err != nil {
		return nil, fmt.Errorf("sign login code: %w", err)
	}
	signatureHex := hexutil.Encode(signature)
	_, refreshToken, err := apiClient.Login(signatureHex, addr, nonce)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	playerIDStr, loggedIn, err := apiClient.IsUserLoggedIn(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("is user logged in: %w", err)
	}
	if !loggedIn {
		return nil, fmt.Errorf("new wallet did not become logged in")
	}
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse player id %q: %w", playerIDStr, err)
	}
	metaRaw, _ := json.Marshal(map[string]any{
		"provision_method": "http_api_login",
		"created_at_unix":  time.Now().Unix(),
	})
	account := &dao.BotAccount{
		PlayerID:       playerID,
		TempPrivateKey: w.GetPrivateKeyHex(),
		TempAddress:    addr,
		Metadata:       string(metaRaw),
	}
	if err := db.InsertBotAccount(account); err != nil {
		return nil, fmt.Errorf("insert bot account: %w", err)
	}
	log.Infow("provisioned_bot_account", "player_id", playerID, "temp_address", addr)
	return account, nil
}

func (s *Service) runBots() {
	log.Infow("run bots", types.ToJsonLoggable(s.addresses))
	for _, b := range s.bots {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			err := b.run()
			if err != nil {
				log.Errorw("cannot run bot", "err", err, "addr", b.addr.String())
			}
		}()
	}
}

func (s *Service) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	registered := s.registered
	s.registered = false
	heartbeatCancel := s.heartbeatCancel
	s.heartbeatCancel = nil
	s.mu.Unlock()

	if heartbeatCancel != nil {
		heartbeatCancel()
	}
	s.heartbeatWG.Wait()
	s.ccl()
	s.wg.Wait()
	if registered {
		typed := make([]types.PlayerAddress, 0, len(s.addresses))
		for _, a := range s.addresses {
			if a != nil {
				typed = append(typed, *a)
			}
		}
		if err := s.registry.MarkStopping(typed); err != nil {
			log.Errorw("bot_mark_stopping_failed", "err", err, "count", len(s.addresses))
		} else {
			log.Infow("bot_mark_stopping_success", "count", len(s.addresses), "marker", 1)
		}
	}
}

func (s *Service) upsertBotsAliveNow() error {
	typed := make([]types.PlayerAddress, 0, len(s.addresses))
	for _, a := range s.addresses {
		if a == nil {
			continue
		}
		typed = append(typed, *a)
	}
	return s.registry.UpsertAlive(time.Now().UnixMilli(), typed)
}

func (s *Service) heartbeatBotsNow() error {
	typed := make([]types.PlayerAddress, 0, len(s.addresses))
	for _, a := range s.addresses {
		if a == nil {
			continue
		}
		typed = append(typed, *a)
	}
	return s.registry.Heartbeat(time.Now().UnixMilli(), typed)
}

func (s *Service) startHeartbeatLoopLocked() {
	if s.registry == nil {
		return
	}
	if s.heartbeatInterval <= 0 {
		s.heartbeatInterval = 5 * time.Second
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.heartbeatCancel = cancel
	s.heartbeatWG.Add(1)
	go func() {
		defer s.heartbeatWG.Done()
		ticker := time.NewTicker(s.heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.heartbeatBotsNow(); err != nil {
					log.Errorw("bot_heartbeat_failed", "err", err)
				}
			}
		}
	}()
}
