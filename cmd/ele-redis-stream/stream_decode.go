package main

import (
	"fmt"

	pb "github.com/CryptoElementals/common/rpc/proto"
	gproto "google.golang.org/protobuf/proto"
)

const (
	StreamRoomEvents  = "room_events"
	StreamLobbyEvents = "lobby_events"
)

// decodeStreamPayload decodes Redis stream entry bytes. Lobby publishes raw pubsub.Event; room/produce often use wrapped Message.
func decodeStreamPayload(payload []byte, kind string) (wrap *pb.Message, bare *pb.Event, err error) {
	switch kind {
	case "event":
		var e pb.Event
		if err := gproto.Unmarshal(payload, &e); err != nil {
			return nil, nil, err
		}
		return nil, &e, nil
	case "message":
		var m pb.Message
		if err := gproto.Unmarshal(payload, &m); err != nil {
			return nil, nil, err
		}
		return &m, nil, nil
	case "auto":
		// Lobby StreamPublisher marshals raw Event bytes. proto.Unmarshal into Message often "succeeds"
		// but leaves an empty Message (unknown fields), so we must pick by populated fields.
		var m pb.Message
		mErr := gproto.Unmarshal(payload, &m)
		var e pb.Event
		eErr := gproto.Unmarshal(payload, &e)
		mOK := mErr == nil && messageLooksPopulated(&m)
		eOK := eErr == nil && eventLooksPopulated(&e)
		if mOK && eOK {
			if m.GetEvent() != nil {
				return &m, nil, nil
			}
			return nil, &e, nil
		}
		if mOK {
			return &m, nil, nil
		}
		if eOK {
			return nil, &e, nil
		}
		return nil, nil, fmt.Errorf("not a populated pubsub.Message (%v) nor pubsub.Event (%v)", mErr, eErr)
	default:
		return nil, nil, fmt.Errorf("invalid --payload %q (use auto, message, event)", kind)
	}
}

func messageLooksPopulated(m *pb.Message) bool {
	if m == nil {
		return false
	}
	if m.GetEvent() != nil {
		return true
	}
	if m.MessageId != "" || m.Topic != "" || m.PublisherId != "" {
		return true
	}
	if m.Timestamp != 0 || len(m.Metadata) > 0 {
		return true
	}
	return false
}

func eventLooksPopulated(e *pb.Event) bool {
	if e == nil {
		return false
	}
	if e.Type != pb.EventType_TYPE_KNOWN {
		return true
	}
	if e.MessageId != "" || len(e.Receivers) > 0 {
		return true
	}
	r := e.ProtoReflect()
	for i := 0; i < r.Descriptor().Oneofs().Len(); i++ {
		if r.WhichOneof(r.Descriptor().Oneofs().Get(i)) != nil {
			return true
		}
	}
	return false
}

func summarizeTournamentMatchOutcome(o *pb.TournamentMatchOutcome) string {
	if o == nil {
		return ""
	}
	next := o.GetNextRoundNo()
	nextStr := fmt.Sprintf("%d", next)
	if o.NextRoundNo == nil {
		nextStr = "(unset)"
	}
	return fmt.Sprintf("TournamentMatchOutcome game_id=%d tournament_id=%s round=%d match=%d round_finished=%v tournament_finished=%v next_round_no=%s",
		o.GetGameId(), o.GetTournamentId(), o.GetRoundNo(), o.GetMatchNo(),
		o.GetRoundFinished(), o.GetTournamentFinished(), nextStr)
}
