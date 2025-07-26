package scanner

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/cmd/ele-scanner/blockchain"
	"github.com/CryptoElementals/common/config"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	eleClient "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	dialTimeout              = 5
	RoomCreatedEventName     = "RoomCreated"
	SubmitCardsHashEventName = "submitCardsHash"
	SubmitCardsEventName     = "submitCards"
)

type eventSigHashCache struct {
	mu              sync.RWMutex
	eventNameToHash map[string]common.Hash //event name to hash
	eventHashToName map[common.Hash]string // event hash to event name
}

// Scanner encapsulates the state and logic for block catching up
type Scanner struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	gethWsRpc            string
	gethHttpRpc          string
	roomServerHttpRpc    string
	gethClient           *ethclient.Client
	rpcClient            *eleClient.RpcClient
	roomManagerAddress   string
	roomManagerAbi       *abi.ABI
	roomAbi              *abi.ABI
	currentScannedHeight uint64
	headNumberOnChain    uint64
	eventSigHashCache    eventSigHashCache // 封装后的缓存
}

// NewScanner creates a new Scanner with its own cancellable context.
func NewScanner(parentCtx context.Context, gethWsRpc string, gethHttpRpc string, roomServerHttpRpc string, roomManagerAddress string, roomManagerAbi *abi.ABI, roomAbi *abi.ABI) *Scanner {
	log.Infof("NewScanner gethWsRpc: %s, gethHttpRpc: %s, roomServerHttpRpc: %s", gethWsRpc, gethHttpRpc, roomServerHttpRpc)
	ctx, cancel := context.WithCancel(parentCtx)
	return &Scanner{
		ctx:                ctx,
		cancel:             cancel,
		gethWsRpc:          gethWsRpc,
		gethHttpRpc:        gethHttpRpc,
		roomServerHttpRpc:  roomServerHttpRpc,
		roomManagerAddress: roomManagerAddress,
		roomManagerAbi:     roomManagerAbi,
		roomAbi:            roomAbi,
		eventSigHashCache: eventSigHashCache{
			eventNameToHash: make(map[string]common.Hash),
			eventHashToName: make(map[common.Hash]string),
		},
	}
}

// Stop gracefully stops the scanner and all its goroutines.
func (s *Scanner) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	log.Info("Scanner Stop() called")
}

func (s *Scanner) Run() {
	var err error
	err = s.initEventSigHashCache()
	if err != nil {
		log.Errorf("initEventSigHashCache failed!!!, err %s", err.Error())
		return
	}

	for {
		syncs, err := db.FindBlockSyncs()
		if err != nil {
			log.Errorf("db.FindBlockSyncs() failed, err %s", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}
		for _, sync := range syncs {
			if sync.Type == "head" {
				s.headNumberOnChain = sync.BlockHeight
				s.currentScannedHeight = sync.BlockHeight + 1
			}
		}
		break
	}

	for {
		s.rpcClient, err = eleClient.NewRpcClient(s.roomServerHttpRpc)
		if err != nil {
			log.Errorf("Failed to create rpcClient to roomServer: %v, retrying in %d seconds...", err.Error(), dialTimeout)
			time.Sleep(time.Duration(dialTimeout) * time.Second)
			continue
		}
		break
	}

	go s.RunCatchUp()
}

func (s *Scanner) RunCatchUp() {
	var err error
	var catchupCancel context.CancelFunc
	for {
		s.gethClient, err = ethclient.Dial(s.gethWsRpc)
		if err != nil {
			log.Errorf("Failed to connect to WebSocket RPC: %v, retrying in %d seconds...", err.Error(), dialTimeout)
			time.Sleep(time.Duration(dialTimeout) * time.Second)
			continue
		}
		log.Info("WebSocket connected, subscribing to new blocks...")

		if catchupCancel != nil {
			catchupCancel() // stop old goroutine
		}
		_, cancel := context.WithCancel(s.ctx)
		catchupCancel = cancel
		go s.CatchUpChain()

		headers := make(chan *types.Header)
		sub, err := s.gethClient.SubscribeNewHead(s.ctx, headers)
		if err != nil {
			log.Infof("Failed to subscribe to new blocks: %v, retrying in %d seconds...", err.Error(), dialTimeout)
			s.gethClient.Close()
			time.Sleep(time.Duration(dialTimeout) * time.Second)
			continue
		}

		for {
			select {
			case <-s.ctx.Done():
				log.Info("Scanner context done, RunCatchUp for headNumberOnChain exited...")
				return
			case err := <-sub.Err():
				log.Infof("Subscription error: %v, reconnecting in %d seconds...", err.Error(), dialTimeout)
				sub.Unsubscribe()
				s.gethClient.Close()
				time.Sleep(time.Duration(dialTimeout) * time.Second)
				goto RECONNECT
			case header := <-headers:
				headNumberOnChain := header.Number.Uint64()
				s.SetHeadNumberOnChain(headNumberOnChain)
				log.Debugf("HeadNumberOnChain is %d", headNumberOnChain)
			}
		}
	RECONNECT:
		// Next reconnect loop
	}
}

func (s *Scanner) SetHeadNumberOnChain(height uint64) {
	s.headNumberOnChain = height
}

func (s *Scanner) CatchUpChain() {
	for {
		select {
		case <-s.ctx.Done():
			s.rpcClient.Close()
			log.Info("Scanner context done, CatchUpChain exited...")
			return
		default:
			if s.currentScannedHeight > s.headNumberOnChain {
				time.Sleep(time.Millisecond * 200)
				continue
			}
			err := s.getAndProcessBlock(big.NewInt(int64(s.currentScannedHeight)))
			if err != nil {
				log.Warnf("catchUpChain goroutine parse block err %v", err.Error())
				time.Sleep(time.Second * 5)
				continue
			}
			err = db.SaveBlockSync(dao.BlockSync{Type: "head", BlockHeight: s.currentScannedHeight})
			if err != nil {
				log.Errorf("insert head block sync to db err %v", err.Error())
				time.Sleep(time.Second * 5)
				continue
			}
			log.Infof("block %d handled successfully", s.currentScannedHeight)
			s.currentScannedHeight++
		}
	}
}

func (s *Scanner) getAndProcessBlock(blockHeight *big.Int) error {
	block, err := blockchain.GetOptimismBlockByNumber(s.ctx, s.gethHttpRpc, blockHeight)
	if err != nil {
		log.Errorf("getBlockByNumber failed, err %s", err.Error())
		return err
	}
	//log.Debugf("blockHeight: %d, block: %+v", blockHeight.Uint64(), block)
	parsedTxs, err := blockchain.ParseOptimismTransactions(block.Transactions)
	if err != nil {
		log.Errorf("ParseOptimismTransactions failed, err %s", err.Error())
		return err
	}
	if len(parsedTxs) != len(block.Transactions) {
		log.Errorf("Parsed tx count %d does not match raw tx count %d", len(parsedTxs), len(block.Transactions))
		return err
	}

	txsToSubmit := make([]*proto.Transaction, 0)

	for _, tx := range parsedTxs {
		if tx.To != "0x4200000000000000000000000000000000000015" {
			log.Debugf("parsed tx: %+v", tx)
		}
		// if strings.EqualFold(tx.To, s.roomManagerAddress) {
		// 	log.Debugf("room manager contract tx: %+v", tx)
		// 	roomCreatedTx, err := processCreateRoomTx(s.ctx, s.gethClient, tx, s.roomManagerAbi)
		// 	if err != nil {
		// 		log.Errorf("processCreateRoomTx failed, err %s, tx %+v", err.Error(), tx)
		// 		return fmt.Errorf("processCreateRoomTx failed, err %s, tx %+v", err.Error(), tx)
		// 	}
		// 	log.Debugf("room created tx: %+v", roomCreatedTx)
		// 	txsToSubmit = append(txsToSubmit, &proto.Transaction{
		// 		TxHash: roomCreatedTx.TxHash.Bytes(),
		// 		Tx: &proto.Transaction_RoomContractCreated{
		// 			RoomContractCreated: &proto.TxRoomContractCreated{
		// 				RoomContractAddress: roomCreatedTx.RoomCreatedEvent.RoomAddress.Hex(),
		// 			},
		// 		},
		// 	})
		// }

		txs, err := s.processTx(tx)
		if err != nil {
			log.Errorf("processTx failed, err %s, tx %+v", err.Error(), tx)
			return err
		}
		txsToSubmit = append(txsToSubmit, txs...)
	}

	if len(txsToSubmit) > 0 && !config.ScannerGConf.RoomServerMocked {
		timeStamp, _ := strconv.ParseUint(block.Timestamp, 0, 64)
		blockNumber, _ := strconv.ParseUint(block.Number, 0, 64)
		err = s.rpcClient.SubmitTransactions(s.ctx, &proto.TransactionBatch{
			BlockHash:    common.HexToHash(block.Hash).Bytes(),
			Timestamp:    timeStamp,
			BlockNumber:  blockNumber,
			Transactions: txsToSubmit,
		})
		if err != nil {
			log.Errorf("submit transactions to roomServer failed, err %s, BlockNumber %d, BlockHash %s, Timestamp %d",
				err.Error(), blockNumber, block.Hash, timeStamp)
			for i, tx := range txsToSubmit {
				jsonStr, _ := protojson.Marshal(tx)
				log.Errorf("txsToSubmit[%d]: %s, txHash(hex): %x", i, string(jsonStr), tx.TxHash)
			}
			return err
		} else {
			log.Infof("submit transactions to roomServer success, BlockNumber %d, BlockHash %s, Timestamp %d",
				blockNumber, block.Hash, timeStamp)
			for i, tx := range txsToSubmit {
				jsonStr, _ := protojson.Marshal(tx)
				log.Debugf("txsToSubmit[%d]: %s, txHash(hex): %x", i, string(jsonStr), tx.TxHash)
			}
		}
	}
	return nil
}

func (s *Scanner) processTx(tx blockchain.OptimismTx) ([]*proto.Transaction, error) {
	txsToSubmit := make([]*proto.Transaction, 0)
	if strings.EqualFold(tx.To, "0x4200000000000000000000000000000000000015") { // specail tx
		return nil, nil
	}

	// todo: filter room manager address after 7702

	// find room tx
	hash := common.HexToHash(tx.Hash)
	receipt, err := s.gethClient.TransactionReceipt(s.ctx, hash)
	if err != nil {
		return nil, err
	}

	for _, vLog := range receipt.Logs {
		s.eventSigHashCache.mu.RLock()
		eventName, ok := s.eventSigHashCache.eventHashToName[vLog.Topics[0]]
		s.eventSigHashCache.mu.RUnlock()
		if !ok {
			continue
		}

		var txSubmit *proto.Transaction
		if eventName == RoomCreatedEventName {
			eventData := contract.RoomManagerContractRoomCreated{}
			if err := s.roomManagerAbi.UnpackIntoInterface(&eventData, eventName, vLog.Data); err != nil {
				return nil, err
			}

			txSubmit = &proto.Transaction{
				TxHash: common.HexToHash(tx.Hash).Bytes(),
				Tx: &proto.Transaction_RoomContractCreated{
					RoomContractCreated: &proto.TxRoomContractCreated{
						RoomContractAddress: eventData.RoomAddress.Hex(),
					},
				},
			}
		}

		if eventName == SubmitCardsHashEventName {
			eventData := contract.RoomContractSubmitCardsHash{}
			if err := s.roomAbi.UnpackIntoInterface(&eventData, eventName, vLog.Data); err != nil {
				return nil, err
			}
			txSubmit = &proto.Transaction{
				TxHash: common.HexToHash(tx.Hash).Bytes(),
				Tx: &proto.Transaction_CommitmentsOnChain{
					CommitmentsOnChain: &proto.TxCommitmentsOnChain{
						RoomContractAddress: tx.To,
						RoundNumber:         uint32(eventData.Round.Uint64()),
						Address: &proto.PlayerAddress{
							//WalletAddress:    eventData.Address.Hex(),
							TemporaryAddress: eventData.Arg0.Hex(),
						},
						Commitment: eventData.CardsHash[:],
					},
				},
			}
		}

		if eventName == SubmitCardsEventName {
			eventData := contract.RoomContractSubmitCards{}
			if err := s.roomAbi.UnpackIntoInterface(&eventData, eventName, vLog.Data); err != nil {
				return nil, err
			}
			cardStrs := strings.Split(eventData.Cards, ",")
			cards := make([]uint32, 0, len(cardStrs))
			for _, s := range cardStrs {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				n, err := strconv.ParseUint(s, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid card value: %v", err)
				}
				cards = append(cards, uint32(n))
			}

			txSubmit = &proto.Transaction{
				TxHash: common.HexToHash(tx.Hash).Bytes(),
				Tx: &proto.Transaction_CardsOnChain{
					CardsOnChain: &proto.TxCardsOnChain{
						RoomContractAddress: tx.To,
						Address: &proto.PlayerAddress{
							//WalletAddress:    eventData.Address.Hex(),
							TemporaryAddress: eventData.Arg0.Hex(),
						},
						RoundNumber: uint32(eventData.Round.Uint64()),
						Salt:        []byte("mockedSalt"),
						Cards:       cards,
					},
				},
			}
		}

		if txSubmit != nil {
			txsToSubmit = append(txsToSubmit, txSubmit)
		}
	}
	return txsToSubmit, nil
}

func (s *Scanner) initEventSigHashCache() error {
	cache := &s.eventSigHashCache
	roomEventNames := []string{SubmitCardsHashEventName, SubmitCardsEventName}
	roomManagerEventNames := []string{RoomCreatedEventName}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	for _, name := range roomEventNames {
		event, ok := s.roomAbi.Events[name]
		if !ok {
			return fmt.Errorf("event %s not found in room ABI", name)
		}
		cache.eventNameToHash[name] = event.ID
		cache.eventHashToName[event.ID] = name
	}

	for _, name := range roomManagerEventNames {
		event, ok := s.roomManagerAbi.Events[name]
		if !ok {
			return fmt.Errorf("event %s not found in roomManager ABI", name)
		}
		cache.eventNameToHash[name] = event.ID
		cache.eventHashToName[event.ID] = name
	}

	return nil
}

func processCreateRoomTx(ctx context.Context, gethClient *ethclient.Client, tx blockchain.OptimismTx, roomManagerAbi *abi.ABI) (*blockchain.RoomCreatedTx, error) {
	hash := common.HexToHash(tx.Hash)
	receipt, err := gethClient.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	roomCreatedTx, err := blockchain.ParseRoomCreatedEvent(receipt, roomManagerAbi)
	if err != nil {
		return nil, err
	}

	return roomCreatedTx, nil
}
