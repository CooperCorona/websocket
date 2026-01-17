package websocket

import (
	"encoding/json"

	"github.com/samber/ro"
)

// we still need a Socket to represent input AND output. Maybe we will want a Hub to
// abstract broadcasting. But we can dispense with the channel and just use a Subject[AnyEvent].
// Then use this custom function to filter for event names and get the strongly typed version
// in response.
func Listen[T any](eventName string) func(ro.Observable[AnyEvent]) ro.Observable[T] {
	return func(input ro.Observable[AnyEvent]) ro.Observable[T] {
		return ro.Pipe3(input,
			ro.Map(func(e AnyEvent) *T {
				if e.Name != eventName {
					return nil
				}
				if t, ok := e.Data.(T); ok {
					return &t
				} else if j, ok := e.Data.(json.RawMessage); ok {
					var d T
					err := json.Unmarshal(j, &d)
					if err != nil {
						return nil
					}
					return &d
				} else {
					// must be some other type. No way to know if the type was intentional or not,
					// so we return nil and stop processing.
					return nil
				}
			}),
			ro.Filter(func(t *T) bool {
				return t != nil
			}),
			ro.Map(func(t *T) T {
				// t is guaranteed to be non-nil by the time we reach here.
				return *t
			}),
		)
	}
}

// we still need a Socket to represent input AND output. Maybe we will want a Hub to
// abstract broadcasting. But we can dispense with the channel and just use a Subject[AnyEvent].
// Then use this custom function to filter for event names and get the strongly typed version
// in response.
func ListenSocket[T any](eventName string) func(ro.Observable[AnySocketEvent]) ro.Observable[SocketEvent[T]] {
	return func(input ro.Observable[AnySocketEvent]) ro.Observable[SocketEvent[T]] {
		return ro.Pipe3(input,
			ro.Map(func(e AnySocketEvent) *SocketEvent[T] {
				if e.Name != eventName {
					return nil
				}
				if t, ok := e.Data.(T); ok {
					return &SocketEvent[T]{
						Name:     e.Name,
						Data:     t,
						Socket:   e.Socket,
						UserInfo: e.UserInfo,
					}
				} else if j, ok := e.Data.(json.RawMessage); ok {
					var d T
					err := json.Unmarshal(j, &d)
					if err != nil {
						return nil
					}
					return &SocketEvent[T]{
						Name:     e.Name,
						Data:     d,
						Socket:   e.Socket,
						UserInfo: e.UserInfo,
					}
				} else {
					// must be some other type. No way to know if the type was intentional or not,
					// so we return nil and stop processing.
					return nil
				}
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

func Publish[T any](name string, observable ro.Observable[AnyEvent], output Socket) ro.Subscription {
	return ro.Pipe1(observable, Listen[T](name)).Subscribe(ro.OnNext(func(t T) {
		output.Send()
	}))
}
