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
	StartANewRoundName       = "startANewRound"
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
		s.rpcClient, err = eleClient.NewRpcClientWithAddr(s.roomServerHttpRpc)
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
				if headNumberOnChain%10 == 0 {
					log.Debugf("HeadNumberOnChain is %d", headNumberOnChain)
				}
			}
		}
	RECONNECT:
		// Next reconnect loop
	}
}

func (s *Scanner) SetHeadNumberOnChain(height uint64) {
	s.headNumberOnChain = height
}

// 新增：有序的交易提交结构
type orderedTxBatch struct {
	blockNumber uint64
	batch       *proto.TransactionBatch
	done        chan error
}

func (s *Scanner) CatchUpChain() {
	const maxWorkers = 5
	blockChan := make(chan uint64, maxWorkers*2)
	submitChan := make(chan *orderedTxBatch, 100)

	// 启动 worker goroutines
	for i := 0; i < maxWorkers; i++ {
		go s.blockWorker(blockChan, submitChan)
	}
	// 启动专门的提交 goroutine（保证顺序）
	go s.orderedSubmitWorker(submitChan)

	for {
		select {
		case <-s.ctx.Done():
			close(blockChan)
			close(submitChan)
			s.rpcClient.Close()
			log.Info("Scanner context done, CatchUpChain exited...")
			return
		default:
			if s.currentScannedHeight > s.headNumberOnChain {
				time.Sleep(time.Millisecond * 200)
				continue
			}
			log.Infof("Start processing block %d", s.currentScannedHeight)
			blockChan <- s.currentScannedHeight
			s.currentScannedHeight++
		}
	}
}

// blockWorker 处理单个区块，但不直接提交
func (s *Scanner) blockWorker(blockChan <-chan uint64, submitChan chan<- *orderedTxBatch) {
	for blockHeight := range blockChan {
		batch, err := s.getAndProcessBlockToBatch(big.NewInt(int64(blockHeight)))
		if err != nil {
			log.Warnf("blockWorker parse block %d err %v", blockHeight, err.Error())
			time.Sleep(time.Second * 5)
			continue // 出错时不保存同步状态，下次重试
		}

		// 如果有交易需要提交
		if batch != nil && len(batch.Transactions) > 0 {
			orderedBatch := &orderedTxBatch{
				blockNumber: blockHeight,
				batch:       batch,
				done:        make(chan error, 1),
			}
			submitChan <- orderedBatch
			// 等待提交完成
			if err := <-orderedBatch.done; err != nil {
				log.Errorf("submit failed for block %d: %v", blockHeight, err)
				continue // 提交失败时不保存同步状态，下次重试
			}
		}

		// 只有处理成功且提交成功（如果有交易）才保存区块同步状态
		err = db.SaveBlockSync(dao.BlockSync{Type: "head", BlockHeight: blockHeight})
		if err != nil {
			log.Errorf("Insert head block sync to db err %v", err.Error())
			time.Sleep(time.Second * 5)
			continue // 保存失败时也不继续，下次重试
		}
		log.Infof("Block %d handled successfully", blockHeight)
	}
}

// orderedSubmitWorker 按顺序提交交易到 roomServer
func (s *Scanner) orderedSubmitWorker(submitChan <-chan *orderedTxBatch) {
	const maxPending = 10000 // 最大缓存阈值，可根据实际情况调整
	expectedBlockNumber := s.currentScannedHeight
	pendingBatches := make(map[uint64]*orderedTxBatch)
	for batch := range submitChan {
		if batch.blockNumber == expectedBlockNumber {
			err := s.submitBatch(batch.batch)
			batch.done <- err
			expectedBlockNumber++
			for {
				if nextBatch, exists := pendingBatches[expectedBlockNumber]; exists {
					err := s.submitBatch(nextBatch.batch)
					nextBatch.done <- err
					delete(pendingBatches, expectedBlockNumber)
					expectedBlockNumber++
				} else {
					break
				}
			}
		} else {
			if len(pendingBatches) >= maxPending {
				// 缓存过大时，拒绝新的 batch，等待处理完成
				log.Errorf("pendingBatches size exceeded: %d, rejecting block %d, waiting for processing", len(pendingBatches), batch.blockNumber)
				batch.done <- fmt.Errorf("pending batches overflow, block %d rejected", batch.blockNumber)
				continue
			}
			pendingBatches[batch.blockNumber] = batch
		}
	}
}

// submitBatch 提交交易批次到 roomServer
func (s *Scanner) submitBatch(batch *proto.TransactionBatch) error {
	if config.ScannerGConf.RoomServerMocked {
		log.Debugf("RoomServer mocked, skipping submit for block %d", batch.BlockNumber)
		return nil
	}
	err := s.rpcClient.SubmitTransactions(s.ctx, batch)
	if err != nil {
		log.Errorf("submit transactions to roomServer failed, err %s, BlockNumber %d, BlockHash %x, Timestamp %d",
			err.Error(), batch.BlockNumber, batch.BlockHash, batch.Timestamp)
		for i, tx := range batch.Transactions {
			jsonStr, _ := protojson.Marshal(tx)
			log.Errorf("txsToSubmit[%d]: %s, txHash(hex): %x", i, string(jsonStr), tx.TxHash)
		}
		return err
	}
	log.Infof("submit transactions to roomServer success, BlockNumber %d, BlockHash %x, Timestamp %d",
		batch.BlockNumber, batch.BlockHash, batch.Timestamp)
	for i, tx := range batch.Transactions {
		jsonStr, _ := protojson.Marshal(tx)
		log.Debugf("txsToSubmit[%d]: %s, txHash(hex): %x", i, string(jsonStr), tx.TxHash)
	}
	return nil
}

// getAndProcessBlockToBatch 返回 TransactionBatch，不直接提交
func (s *Scanner) getAndProcessBlockToBatch(blockHeight *big.Int) (*proto.TransactionBatch, error) {
	block, err := blockchain.GetOptimismBlockByNumber(s.ctx, s.gethHttpRpc, blockHeight)
	if err != nil {
		log.Errorf("getBlockByNumber failed, err %s", err.Error())
		return nil, err
	}
	parsedTxs, err := blockchain.ParseOptimismTransactions(block.Transactions)
	if err != nil {
		log.Errorf("ParseOptimismTransactions failed, err %s", err.Error())
		return nil, err
	}
	if len(parsedTxs) != len(block.Transactions) {
		log.Errorf("Parsed tx count %d does not match raw tx count %d", len(parsedTxs), len(block.Transactions))
		return nil, err
	}
	txsToSubmit := make([]*proto.Transaction, 0)
	for _, tx := range parsedTxs {
		if tx.To != "0x4200000000000000000000000000000000000015" {
			log.Debugf("parsed tx: %+v", tx)
		}
		txs, err := s.processTx(tx)
		if err != nil {
			log.Errorf("processTx failed, err %s, tx %+v", err.Error(), tx)
			return nil, err
		}
		txsToSubmit = append(txsToSubmit, txs...)
	}
	if len(txsToSubmit) > 0 {
		timeStamp, _ := strconv.ParseUint(block.Timestamp, 0, 64)
		blockNumber, _ := strconv.ParseUint(block.Number, 0, 64)
		return &proto.TransactionBatch{
			BlockHash:    common.HexToHash(block.Hash).Bytes(),
			Timestamp:    timeStamp,
			BlockNumber:  blockNumber,
			Transactions: txsToSubmit,
		}, nil
	}
	return nil, nil
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
							TemporaryAddress: eventData.Player.Hex(),
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
							TemporaryAddress: eventData.Player.Hex(),
						},
						RoundNumber: uint32(eventData.Round.Uint64()),
						Salt:        []byte(eventData.Salt),
						Cards:       cards,
					},
				},
			}
		}

		if eventName == StartANewRoundName {
			eventData := contract.RoomContractStartANewRound{}
			if err := s.roomAbi.UnpackIntoInterface(&eventData, eventName, vLog.Data); err != nil {
				return nil, err
			}
			txSubmit = &proto.Transaction{
				TxHash: common.HexToHash(tx.Hash).Bytes(),
				Tx: &proto.Transaction_RoomContractSetupReady{
					RoomContractSetupReady: &proto.TxRoomContractRoundSetupReady{
						RoomContractAddress: tx.To,
						RoundNumber:         uint32(eventData.Round.Uint64()),
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
	roomEventNames := []string{SubmitCardsHashEventName, SubmitCardsEventName, StartANewRoundName}
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
