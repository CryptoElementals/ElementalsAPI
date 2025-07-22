package types

import "errors"

type JoinQueueEvent struct {
	PlayerAddress PlayerAddress
}

type ExitQueueEvent struct {
	PlayerAddress PlayerAddress
}

var MatchFailedError = errors.New("match failed")
