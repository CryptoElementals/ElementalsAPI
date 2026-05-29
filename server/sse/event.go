package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// EventType identifies the kind of SSE payload.
type EventType string

const (
	EventTypeNotification       EventType = "notification"
	EventTypeDataChange         EventType = "data_change"
	EventTypeTournamentSnapshot EventType = "tournament_snapshot"
	EventTypeStatusUpdate       EventType = "status_update"
	EventTypeError              EventType = "error"
	EventTypeHeartbeat          EventType = "heartbeat"
)

// Event is the JSON payload sent over Server-Sent Events.
type Event struct {
	Type        EventType              `json:"type"`
	Data        interface{}            `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	RequestUUID string                 `json:"RequestUUID,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Write encodes event as an SSE data frame and flushes the response.
func Write(writer http.ResponseWriter, flusher http.Flusher, event Event) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err = writer.Write([]byte(fmt.Sprintf("data: %s\n\n", jsonData))); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
