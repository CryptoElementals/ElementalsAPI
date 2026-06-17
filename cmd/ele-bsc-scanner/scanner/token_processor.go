package scanner

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/CryptoElementals/common/internal/evmrpc"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	depositedEventName = "Deposited"
	withdrawnEventName = "Withdrawn"
	eventTypeDeposit   = "deposit"
	eventTypeWithdraw  = "withdraw"
)

type TokenProcessor struct {
	chainID           uint64
	registry          *WalletRegistry
	tokenCollectorABI *abi.ABI
	depositedTopic    common.Hash
	withdrawnTopic    common.Hash
}

func NewTokenProcessor(chainID uint64, registry *WalletRegistry) (*TokenProcessor, error) {
	parsed, err := abi.JSON(strings.NewReader(contract.TokenCollectorContractABI))
	if err != nil {
		return nil, fmt.Errorf("parse TokenCollector ABI: %w", err)
	}
	deposited, ok := parsed.Events[depositedEventName]
	if !ok {
		return nil, fmt.Errorf("event %s not in ABI", depositedEventName)
	}
	withdrawn, ok := parsed.Events[withdrawnEventName]
	if !ok {
		return nil, fmt.Errorf("event %s not in ABI", withdrawnEventName)
	}
	return &TokenProcessor{
		chainID:           chainID,
		registry:          registry,
		tokenCollectorABI: &parsed,
		depositedTopic:    deposited.ID,
		withdrawnTopic:    withdrawn.ID,
	}, nil
}

func (p *TokenProcessor) ParseEvents(data *BlockData) []TokenCollectorEvent {
	if data == nil {
		return nil
	}
	txByHash := make(map[string]evmrpc.Tx, len(data.Txs))
	for _, tx := range data.Txs {
		txByHash[strings.ToLower(tx.Hash)] = tx
	}

	var events []TokenCollectorEvent
	for _, lg := range data.Logs {
		addr := common.HexToAddress(lg.Address)
		if !p.registry.Contains(addr) || len(lg.Topics) == 0 {
			continue
		}
		topic0 := common.HexToHash(lg.Topics[0])
		meta := p.eventMeta(data, lg, addr)
		switch topic0 {
		case p.depositedTopic:
			ev := contract.TokenCollectorContractDeposited{}
			if err := p.tokenCollectorABI.UnpackIntoInterface(&ev, depositedEventName, common.FromHex(lg.Data)); err != nil {
				log.Errorf("unpack Deposited: %v", err)
				continue
			}
			if len(lg.Topics) > 2 {
				ev.From = common.BytesToAddress(common.HexToHash(lg.Topics[1]).Bytes())
				ev.PlayerId = new(big.Int).SetBytes(common.HexToHash(lg.Topics[2]).Bytes())
			}
			meta.EventType = eventTypeDeposit
			meta.Deposit = &DepositPayload{
				PlayerID:    ev.PlayerId.Int64(),
				FromAddress: ev.From.Hex(),
				Amount:      ev.Amount.String(),
				NewCredited: ev.NewCredited.String(),
			}
			events = append(events, meta)
		case p.withdrawnTopic:
			ev := contract.TokenCollectorContractWithdrawn{}
			if err := p.tokenCollectorABI.UnpackIntoInterface(&ev, withdrawnEventName, common.FromHex(lg.Data)); err != nil {
				log.Errorf("unpack Withdrawn: %v", err)
				continue
			}
			if len(lg.Topics) > 2 {
				ev.Operator = common.BytesToAddress(common.HexToHash(lg.Topics[1]).Bytes())
				ev.To = common.BytesToAddress(common.HexToHash(lg.Topics[2]).Bytes())
			}
			var playerID int64
			if tx, ok := txByHash[strings.ToLower(lg.TxHash)]; ok {
				playerID = decodeWithdrawPlayerID(p.tokenCollectorABI, tx.Input)
			}
			meta.EventType = eventTypeWithdraw
			meta.Withdraw = &WithdrawPayload{
				PlayerID:  playerID,
				Operator:  ev.Operator.Hex(),
				ToAddress: ev.To.Hex(),
				Amount:    ev.Amount.String(),
			}
			events = append(events, meta)
		}
	}
	return events
}

func (p *TokenProcessor) eventMeta(data *BlockData, lg evmrpc.ReceiptLog, addr common.Address) TokenCollectorEvent {
	return TokenCollectorEvent{
		ChainID:          p.chainID,
		BlockNumber:      data.BlockNumber,
		BlockHash:        data.BlockHash,
		Timestamp:        data.Timestamp,
		TxHash:           lg.TxHash,
		LogIndex:         lg.LogIndex,
		CollectorAddress: strings.ToLower(addr.Hex()),
	}
}

func decodeWithdrawPlayerID(parsed *abi.ABI, inputHex string) int64 {
	if len(inputHex) < 10 {
		return 0
	}
	input := common.FromHex(inputHex)
	if len(input) < 4 {
		return 0
	}
	method, err := parsed.MethodById(input[:4])
	if err != nil || method.Name != "withdraw" {
		return 0
	}
	vals, err := method.Inputs.Unpack(input[4:])
	if err != nil || len(vals) == 0 {
		return 0
	}
	if pid, ok := vals[0].(*big.Int); ok {
		return pid.Int64()
	}
	return 0
}
