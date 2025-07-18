package testing

import (
	reflect "reflect"

	types "github.com/CryptoElementals/common/worker/types"
	gomock "github.com/golang/mock/gomock"
)

var _ gomock.Matcher = &EventMatcher{}

func NewEventMatcher(eq func(*types.Event) bool) *EventMatcher {
	return &EventMatcher{
		eq: eq,
	}
}

func NewEventTypeMatcher(evt any) *EventMatcher {
	return NewEventMatcher(func(e *types.Event) bool {
		return reflect.TypeOf(e.Data) == reflect.TypeOf(evt)
	})
}

type EventMatcher struct {
	eq func(*types.Event) bool
}

// Matches implements gomock.Matcher.
func (e *EventMatcher) Matches(x interface{}) bool {
	evt, ok := x.(*types.Event)
	if !ok {
		return false
	}
	return e.eq(evt)
}

// String implements gomock.Matcher.
func (e *EventMatcher) String() string {
	return "EventMatcher"
}
