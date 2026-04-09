package pubsub

import (
	"strings"

	"github.com/CryptoElementals/common/rpc/proto"
)

// EventTargetsReceiver reports whether ev is intended for self using Receivers.
// Nil self matches everything. Empty Receivers is treated as broadcast (all subscribers see the event).
func EventTargetsReceiver(ev *proto.Event, self *proto.PlayerAddress) bool {
	if self == nil {
		return true
	}
	if ev == nil {
		return false
	}
	rec := ev.GetReceivers()
	if len(rec) == 0 {
		return true
	}
	for _, r := range rec {
		if r == nil {
			continue
		}
		if r.GetId() != self.GetId() {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(r.GetTemporaryAddress()), strings.TrimSpace(self.GetTemporaryAddress())) {
			return true
		}
	}
	return false
}
