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
	toSubmitHeight       uint64
	toSubmitHeightMutex  sync.RWMutex // 添加同步机制
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
	if s.gethClient != nil {
		s.gethClient.Close()
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
				s.toSubmitHeight = sync.BlockHeight + 1
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
				if headNumberOnChain <= s.headNumberOnChain { // chain reorged, not allowed
					log.Warnf("Chain reorged!!! From %d to %d", s.headNumberOnChain, headNumberOnChain)
				}
				s.SetHeadNumberOnChain(headNumberOnChain)
				if headNumberOnChain%10 == 0 {
					log.Infof("HeadNumberOnChain is %d", headNumberOnChain)
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
	const maxWorkers = 100
	const blockQueueMax = 50
	submitChan := make(chan *orderedTxBatch, 100)

	blockQueue := make(chan uint64, 200)

	// 任务分发协程：基于 toSubmitHeight 控制投递，避免投递过多未处理的区块
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				log.Info("Task distributor context done, exiting...")
				return
			default:
				if s.currentScannedHeight > s.headNumberOnChain {
					time.Sleep(time.Millisecond * 200)
					continue
				}

				if len(blockQueue) <= blockQueueMax {
					select {
					case <-s.ctx.Done():
						log.Info("Task producer context done while sending to blockQueue, exiting...")
						return
					case blockQueue <- s.currentScannedHeight:
						log.Debugf("Task producer add block height %d to blockQueue, blockQueue len %d", s.currentScannedHeight, len(blockQueue))
						s.currentScannedHeight++
					}
				} else {
					// 如果投递的区块太多，等待处理
					log.Debugf("Task producer waiting for blockQueue to be consumned, blockQueue len %d", len(blockQueue))
					time.Sleep(time.Millisecond * 100)
				}
			}
		}
	}()

	// worker 并发消费 blockNumber，失败重试，不跳过
	for i := 0; i < maxWorkers; i++ {
		go func(workerID int) {
			for {
				select {
				case <-s.ctx.Done():
					log.Infof("Worker %d context done, exiting...", workerID)
					return
				case blockNumber, ok := <-blockQueue:
					if !ok {
						log.Infof("Worker %d blockQueue closed, exiting...", workerID)
						return
					}

					log.Debugf("Task consumer Worker %d processing block %d", workerID, blockNumber)
					batch, err := s.getAndProcessBlockToBatch(big.NewInt(int64(blockNumber)))
					if err != nil {
						log.Warnf("Worker %d parse block %d err %v", workerID, blockNumber, err.Error())
						// 失败重试，重新放回队列
						go func(bn uint64) { blockQueue <- bn }(blockNumber)
						time.Sleep(time.Second * 3)
						continue
					}

					log.Debugf("Worker %d sending block %d to submitChan", workerID, blockNumber)
					orderedBatch := &orderedTxBatch{
						blockNumber: blockNumber,
						batch:       batch,
						done:        make(chan error, 1),
					}

					// 使用 select 避免在 context 取消时阻塞
					select {
					case submitChan <- orderedBatch:
					case <-s.ctx.Done():
						log.Infof("Worker %d context done while sending to submitChan, exiting...", workerID)
						return
					}

					// 等待结果，但也要响应 context 取消
					select {
					case err := <-orderedBatch.done:
						if err != nil {
							log.Errorf("Worker %d submit failed for block %d: %v", workerID, blockNumber, err)
							// 失败重试，重新放回队列
							go func(bn uint64) { blockQueue <- bn }(blockNumber)
							time.Sleep(time.Second * 3)
							continue
						}
					case <-s.ctx.Done():
						log.Infof("Worker %d context done while waiting for result, exiting...", workerID)
						return
					}

					if blockNumber%10 == 0 {
						err = db.SaveBlockSync(dao.BlockSync{Type: "head", BlockHeight: blockNumber})
						if err != nil {
							log.Errorf("Worker %d Insert head block sync to db err %v, blockNumber %d. Don't update now!!！", workerID, err.Error(), blockNumber)
							// 失败重试，重新放回队列
							//go func(bn uint64) { blockQueue <- bn }(blockNumber)
							//time.Sleep(time.Second * 5)
						} else {
							s.toSubmitHeightMutex.RLock()
							currentToSubmitHeight := s.toSubmitHeight
							s.toSubmitHeightMutex.RUnlock()
							log.Infof("Worker %d Block %d handled successfully, s.toSubmitHeight %d, s.currentScannedHeight %d", workerID, blockNumber, currentToSubmitHeight, s.currentScannedHeight)
						}
					} else {
						s.toSubmitHeightMutex.RLock()
						currentToSubmitHeight := s.toSubmitHeight
						s.toSubmitHeightMutex.RUnlock()
						log.Infof("Worker %d Block %d handled successfully(not save to db), s.toSubmitHeight %d, s.currentScannedHeight %d", workerID, blockNumber, currentToSubmitHeight, s.currentScannedHeight)
					}
				}
			}
		}(i)
	}
	// 启动专门的提交 goroutine（保证顺序）
	go s.orderedSubmitWorker(submitChan)

	// 主线程只负责监听退出
	<-s.ctx.Done()
	s.rpcClient.Close()
	log.Info("Scanner context done, CatchUpChain exited...")
}

// orderedSubmitWorker 按顺序提交交易到 roomServer
func (s *Scanner) orderedSubmitWorker(submitChan <-chan *orderedTxBatch) {
	pendingBatches := make(map[uint64]*orderedTxBatch)
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()

	for {
		var batch *orderedTxBatch
		var ok bool
		select {
		case <-s.ctx.Done():
			log.Info("Scanner context done, orderedSubmitWorker exited...")
			return
		case batch, ok = <-submitChan:
			if !ok {
				return
			}
			log.Debugf("orderedSubmitWorker received batch for block %d", batch.blockNumber)
			pendingBatches[batch.blockNumber] = batch
			log.Debugf("added block %d to pendingBatches, current size: %d", batch.blockNumber, len(pendingBatches))
		case <-tick.C:
			if len(pendingBatches) > 0 {
				log.Infof("orderedSubmitWorker tick: try to process pendingBatches, toSubmitHeight=%d, pendingBatches size: %d", s.toSubmitHeight, len(pendingBatches))
			} else {
				log.Debugf("orderedSubmitWorker tick: no pendingBatches, toSubmitHeight=%d", s.toSubmitHeight)
				continue
			}
		}

		// 每次有新 batch 或 tick 都尝试推进
		for {
			s.toSubmitHeightMutex.RLock()
			currentToSubmitHeight := s.toSubmitHeight
			s.toSubmitHeightMutex.RUnlock()

			if nextBatch, exists := pendingBatches[currentToSubmitHeight]; exists {
				log.Debugf("processing block %d from pendingBatches", currentToSubmitHeight)
				var err error
				if nextBatch.batch != nil {
					err = s.submitBatch(nextBatch.batch)
				} else {
					err = nil
				}
				// 使用非阻塞方式发送结果，防止死锁
				select {
				case nextBatch.done <- err:
				default:
					log.Warnf("failed to send result to batch %d", currentToSubmitHeight)
				}

				if err != nil {
					log.Errorf("submit failed for block %d, keeping in pendingBatches for retry", currentToSubmitHeight)
					break // 退出循环，等待下次重试
				} else {
					delete(pendingBatches, currentToSubmitHeight)
					s.toSubmitHeightMutex.Lock()
					s.toSubmitHeight++
					newToSubmitHeight := s.toSubmitHeight
					s.toSubmitHeightMutex.Unlock()
					log.Infof("toSubmitHeight advanced to %d, pendingBatches size: %d", newToSubmitHeight, len(pendingBatches))
				}
			} else {
				log.Debugf("waiting for block %d, pendingBatches size: %d", currentToSubmitHeight, len(pendingBatches))
				if len(pendingBatches) > 0 {
					blockNums := make([]uint64, 0, len(pendingBatches))
					for bn := range pendingBatches {
						blockNums = append(blockNums, bn)
					}
					log.Debugf("pendingBatches contains blocks: %v", blockNums)
				}
				break
			}
		}
	}
}

// submitBatch 提交交易批次到 roomServer
func (s *Scanner) submitBatch(batch *proto.TransactionBatch) error {
	if batch == nil {
		// 空 batch，没有交易需要提交
		return nil
	}
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
