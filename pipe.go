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

func ListenAny[U any](eventName string) func(ro.Observable[AnySocketEvent[U]]) ro.Observable[AnySocketEvent[U]] {
	return ro.Filter(func(e AnySocketEvent[U]) bool { return e.Name == eventName })
}

func Listen[T any, U any](eventName string) func(ro.Observable[AnySocketEvent[U]]) ro.Observable[SocketEvent[T, U]] {
	return func(input ro.Observable[AnySocketEvent[U]]) ro.Observable[SocketEvent[T, U]] {
		return ro.Pipe3(input,
			ro.Map(func(e AnySocketEvent[U]) *SocketEvent[T, U] {
				if e.Name != eventName {
					return nil
				}
				socketEvent, err := Cast[T](e)
				if err != nil {
					return nil
				}
				return &socketEvent
			}),
			ro.Filter(func(t *SocketEvent[T, U]) bool {
				return t != nil
			}),
			ro.Map(func(t *SocketEvent[T, U]) SocketEvent[T, U] {
				// t is guaranteed to be non-nil by the time we reach here.
				return *t
			}),
		)
	}
}

func ToAnySocketEvent[U any](socket Socket) func(ro.Observable[AnyEvent]) ro.Observable[AnySocketEvent[U]] {
	return ro.Map(func(e AnyEvent) AnySocketEvent[U] {
		return AnySocketEvent[U]{
			Name:   e.Name,
			Data:   e.Data,
			Socket: socket,
		}
	})
}
