package websocket

import (
	"github.com/samber/ro"
)

type Identifiable interface {
	ID() string
}

func WithID[T Identifiable](id string) func(T) bool {
	return func(t T) bool {
		return t.ID() == id
	}
}

func ListenAny(eventName string) func(ro.Observable[AnySocketEvent]) ro.Observable[AnySocketEvent] {
	return ro.Filter(func(e AnySocketEvent) bool { return e.Name == eventName })
}

func Listen[T any](eventName string) func(ro.Observable[AnySocketEvent]) ro.Observable[SocketEvent[T]] {
	return func(input ro.Observable[AnySocketEvent]) ro.Observable[SocketEvent[T]] {
		return ro.Pipe3(input,
			ro.Map(func(e AnySocketEvent) *SocketEvent[T] {
				if e.Name != eventName {
					return nil
				}
				socketEvent, err := Cast[T](e)
				if err != nil {
					return nil
				}
				return &socketEvent
			}),
			ro.Filter(func(t *SocketEvent[T]) bool {
				return t != nil
			}),
			ro.Map(func(t *SocketEvent[T]) SocketEvent[T] {
				// t is guaranteed to be non-nil by the time we reach here.
				return *t
			}),
		)
	}
}
