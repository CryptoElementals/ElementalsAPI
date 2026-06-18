package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register(LIST_CHAIN_TOKEN_LEDGERS_LABEL, NewListChainTokenLedgersTask, COOKIEAUTH)
}

type ListChainTokenLedgersRequest struct {
	BaseRequest
	PlayerID  string `mapstructure:"PlayerID" validate:"required"`
	EventType string `mapstructure:"EventType"`
	Status    string `mapstructure:"Status"`
	Limit     int32  `mapstructure:"Limit"`
	Offset    int32  `mapstructure:"Offset"`
}

type ListChainTokenLedgersResponse struct {
	BaseResponse
	Records []ChainTokenLedgerRecordDTO `json:"Records"`
	Total   int64                       `json:"Total"`
}

type ChainTokenLedgerRecordDTO struct {
	ID               uint64 `json:"ID"`
	RequestID        string `json:"RequestID"`
	ChainID          int64  `json:"ChainID"`
	TxHash           string `json:"TxHash"`
	LogIndex         uint32 `json:"LogIndex"`
	BlockNumber      uint64 `json:"BlockNumber"`
	BlockHash        string `json:"BlockHash"`
	EventType        string `json:"EventType"`
	PlayerID         int64  `json:"PlayerID"`
	CollectorAddress string `json:"CollectorAddress"`
	AmountWei        string `json:"AmountWei"`
	TokenDelta       int32  `json:"TokenDelta"`
	Status           string `json:"Status"`
	FailReason       string `json:"FailReason"`
	Signature        string `json:"Signature,omitempty"`
	FromAddress      string `json:"FromAddress,omitempty"`
	ToAddress        string `json:"ToAddress,omitempty"`
	Operator         string `json:"Operator,omitempty"`
	NewCreditedWei   string `json:"NewCreditedWei,omitempty"`
	CreatedAt        int64  `json:"CreatedAt"`
}

type ListChainTokenLedgersTask struct {
	Request  *ListChainTokenLedgersRequest
	Response *ListChainTokenLedgersResponse
}

func NewListChainTokenLedgersRequest(data *map[string]interface{}) (*ListChainTokenLedgersRequest, error) {
	req := &ListChainTokenLedgersRequest{}
	if err := mapstructure.Decode(*data, &req); err != nil {
		return nil, err
	}
	req.BaseRequest.RequestUUID = (*data)["RequestUUID"].(string)
	return req, nil
}

func NewListChainTokenLedgersResponse(sessionID string) *ListChainTokenLedgersResponse {
	return &ListChainTokenLedgersResponse{
		BaseResponse: BaseResponse{
			Action:      LIST_CHAIN_TOKEN_LEDGERS_LABEL + "Response",
			RequestUUID: sessionID,
		},
		Records: []ChainTokenLedgerRecordDTO{},
	}
}

func NewListChainTokenLedgersTask(data *map[string]interface{}) (Task, error) {
	req, err := NewListChainTokenLedgersRequest(data)
	if err != nil {
		return nil, err
	}
	task := &ListChainTokenLedgersTask{
		Request:  req,
		Response: NewListChainTokenLedgersResponse(req.BaseRequest.RequestUUID),
	}
	validate := validator.New()
	if err := validate.Struct(task.Request); err != nil {
		return nil, err
	}
	return task, nil
}

func (task *ListChainTokenLedgersTask) Run(c *gin.Context) (Response, error) {
	playerID, err := strconv.ParseInt(strings.TrimSpace(task.Request.PlayerID), 10, 64)
	if err != nil || playerID <= 0 {
		return nil, errors.ParamsJudgeError("invalid player id")
	}

	resp, err := client.ListChainTokenLedgers(context.Background(), ServerTypeFromGin(c), &proto.ListChainTokenLedgersRequest{
		PlayerId:  playerID,
		EventType: strings.TrimSpace(task.Request.EventType),
		Status:    strings.TrimSpace(task.Request.Status),
		Limit:     task.Request.Limit,
		Offset:    task.Request.Offset,
	})
	if err != nil {
		return nil, errors.ActionError(err.Error())
	}

	task.Response.Total = resp.GetTotal()
	for _, rec := range resp.GetRecords() {
		task.Response.Records = append(task.Response.Records, ChainTokenLedgerRecordDTO{
			ID:               rec.GetId(),
			RequestID:        rec.GetRequestId(),
			ChainID:          rec.GetChainId(),
			TxHash:           rec.GetTxHash(),
			LogIndex:         rec.GetLogIndex(),
			BlockNumber:      rec.GetBlockNumber(),
			BlockHash:        rec.GetBlockHash(),
			EventType:        rec.GetEventType(),
			PlayerID:         rec.GetPlayerId(),
			CollectorAddress: rec.GetCollectorAddress(),
			AmountWei:        rec.GetAmountWei(),
			TokenDelta:       rec.GetTokenDelta(),
			Status:           rec.GetStatus(),
			FailReason:       rec.GetFailReason(),
			Signature:        rec.GetSignature(),
			FromAddress:      rec.GetFromAddress(),
			ToAddress:        rec.GetToAddress(),
			Operator:         rec.GetOperator(),
			NewCreditedWei:   rec.GetNewCreditedWei(),
			CreatedAt:        rec.GetCreatedAt(),
		})
	}
	return task.Response, nil
}
