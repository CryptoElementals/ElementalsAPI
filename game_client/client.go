package gameclient

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/events"
)

type GameClient interface {
	ConfirmBattle(ctx context.Context, addr *types.PlayerAddress, gameID uint, roundNumber uint) error
	ContinueGame(ctx context.Context, addr *types.PlayerAddress, gameID uint) error
	ExitQueue(ctx context.Context, addr *types.PlayerAddress) error
	GetBattleInfo(ctx context.Context, gameID uint, roundNumber uint) (BattleInfo, error)
	GetGamePhase(ctx context.Context, addr *types.PlayerAddress) (GamePhase, error)
	JoinQueue(ctx context.Context, addr *types.PlayerAddress) error
	RefuseContinueGame(ctx context.Context, addr *types.PlayerAddress, gameID uint) error
	Subscribe(topic string, subscriberID string, evtChan chan *proto.Event, errChan chan error) error
}

type GamePhaseWrapper struct {
	ProtoGamePhase *proto.GamePhase
	HttpGamePhase  *api.GetGamePhaseResponse
}

func (w *GamePhaseWrapper) GameID() uint {
	if w.ProtoGamePhase != nil {
		return uint(w.ProtoGamePhase.PvPInfo.GameID)
	}
	if w.HttpGamePhase != nil {
		return uint(w.HttpGamePhase.PvPInfo.GameID)
	}
	return 0
}
func (w *GamePhaseWrapper) ContractAddress() string {
	if w.ProtoGamePhase != nil {
		return w.ProtoGamePhase.PvPInfo.ContractAddress
	}
	if w.HttpGamePhase != nil {
		return w.HttpGamePhase.PvPInfo.ContractAddress
	}
	return ""
}

func (w *GamePhaseWrapper) Players() []*types.PlayerAddress {
	if w.ProtoGamePhase != nil {
		players := make([]*types.PlayerAddress, len(w.ProtoGamePhase.Players))
		for i, p := range w.ProtoGamePhase.Players {
			players[i] = &types.PlayerAddress{}
			players[i].FromProto(p.Address)
		}
		return players
	}
	if w.HttpGamePhase != nil {
		players := make([]*types.PlayerAddress, len(w.ProtoGamePhase.Players))
		for i, p := range w.HttpGamePhase.Players {
			players[i] = &types.PlayerAddress{
				WalletAddress: p.Address,
			}
		}
		return players
	}
	return nil
}

type BattleInfoWrapper struct {
	ProtoBattleInfo *proto.GetBattleInfoResponse
	HttpBattleInfo  *api.GetBattleInfoResponse
}

func (w *BattleInfoWrapper) IsGameOver() bool {
	if w.ProtoBattleInfo != nil {
		return w.ProtoBattleInfo.RoundResult.IsGameOver
	}
	if w.HttpBattleInfo != nil {
		return w.HttpBattleInfo.RoundResult.IsGameOver
	}
	return false
}

func (w *BattleInfoWrapper) RoundResult() any {
	if w.ProtoBattleInfo != nil {
		return w.ProtoBattleInfo.RoundResult
	}
	if w.HttpBattleInfo != nil {
		return w.HttpBattleInfo.RoundResult
	}
	return nil
}
func (w *BattleInfoWrapper) GameResult() any {
	if w.ProtoBattleInfo != nil {
		return w.ProtoBattleInfo.GameResult
	}
	if w.HttpBattleInfo != nil {
		return w.HttpBattleInfo.GameResult
	}
	return nil
}

type GamePhase interface {
	GameID() uint
	ContractAddress() string
	Players() []*types.PlayerAddress
}

type BattleInfo interface {
	IsGameOver() bool
	RoundResult() any
	GameResult() any
}

func WrapRpcClient(client *rpc.Client) GameClient {
	return &RpcClientWrapper{Client: client}
}

type RpcClientWrapper struct {
	*rpc.Client
}

func (w *RpcClientWrapper) GetBattleInfo(ctx context.Context, gameID uint, roundNumber uint) (BattleInfo, error) {
	resp, err := w.Client.GetBattleInfo(ctx, gameID, roundNumber)
	if err != nil {
		return nil, err
	}
	return &BattleInfoWrapper{ProtoBattleInfo: resp}, nil
}
func (w *RpcClientWrapper) GetGamePhase(ctx context.Context, addr *types.PlayerAddress) (GamePhase, error) {
	resp, err := w.Client.GetGamePhase(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &GamePhaseWrapper{ProtoGamePhase: resp}, nil
}

func WrapHttpClient(client *HttpClient) GameClient {
	return &HttpClientWrapper{HttpClient: client}
}

type HttpClientWrapper struct {
	*HttpClient
}

func (w *HttpClientWrapper) GetBattleInfo(ctx context.Context, gameID uint, roundNumber uint) (BattleInfo, error) {
	resp, err := w.HttpClient.GetBattleInfo(ctx, gameID, roundNumber)
	if err != nil {
		return nil, err
	}
	return &BattleInfoWrapper{HttpBattleInfo: resp}, nil
}
func (w *HttpClientWrapper) GetGamePhase(ctx context.Context, addr *types.PlayerAddress) (GamePhase, error) {
	resp, err := w.HttpClient.GetGamePhase(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &GamePhaseWrapper{HttpGamePhase: resp}, nil
}

func (w *HttpClientWrapper) Subscribe(topic string, subscriberID string, evtChan chan *proto.Event, errChan chan error) error {
	ch := make(chan *events.Event, 10)
	w.HttpClient.Subscribe(topic, subscriberID, ch)
	for evt := range ch {
		protoEvt, err := parseEvent(evt)
		if err != nil {
			errChan <- err
		} else {
			evtChan <- protoEvt
		}
	}
	return nil
}

func parseEvent(evt *events.Event) (*proto.Event, error) {
	var protoEvt *proto.Event
	switch evt.Type {
	case events.EventTypeError:
		dataMap, ok := evt.Data.(map[string]interface{})
		if !ok {
			return nil, nil
		}
		errStr, ok := dataMap["error"].(string)
		if !ok {
			return nil, nil
		}
		return nil, fmt.Errorf("event error: %s", errStr)
	case events.EventTypeNotification:
		return nil, nil
	case events.EventTypeHeartbeat:
		return nil, nil
	case events.EventTypeStatusUpdate:
		dataMap, ok := evt.Data.(map[string]interface{})
		if !ok {
			return nil, nil
		}
		evtType, ok := dataMap["EventType"].(string)
		if !ok {
			return nil, nil
		}
		switch evtType {
		case "matched":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_MATCHED,
			}
		case "partConfirmed":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_PART_CONFIRMED,
			}
		case "gameCreated":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_GAME_CREATED,
			}
		case "roundReady":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_ROUND_READY,
			}
		case "commitmentsOnChain":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
			}
		case "cardsOnChain":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_CARDS_ON_CHAIN,
			}
		case "roundComplete":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_ROUND_COMPLETE,
			}
		case "gameComplete":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_GAME_COMPLETE,
			}
		case "playerOffline":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_PLAYER_OFFLINE,
			}
		case "continueCanceled":
			protoEvt = &proto.Event{
				Type: proto.EventType_TYPE_CONTINUE_CANCELED,
			}
		case "unknown":
			return nil, nil
		}
	}
	return protoEvt, nil
}
