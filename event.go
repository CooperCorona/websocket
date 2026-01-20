package websocket

import (
	"encoding/json"
	"errors"
)

const (
	SocketConnectEventName = "SocketConnectEvent"
	SocketCloseEventName   = "SocketCloseEventName"
	SocketErrorEventName   = "SocketErrorEventName"
)

var (
	ErrCannotCastType = errors.New("data property of AnySocketEvent is not the given type. cannot cast.")
)

func AlwaysTrue[T any]() func(T) bool {
	return func(t T) bool { return true }
}

// A type-erased event sent to or from a socket.
type AnyEvent struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

// AnySocketEvent represents a type-erased event with a known provenance.
// Socket may be nil, but that represents an event with no
// source, such as a programatically determined one.
type AnySocketEvent struct {
	Name   string
	Data   any
	Socket Socket
}

// SocketEvent is a parameterized event with a known provenance.
// Socket may be nil, but that represents an event with no
// source, such as a programatically determined one.
type SocketEvent[T any] struct {
	Name   string
	Data   T
	Socket Socket
}

type ConnectEvent[T any] struct {
	UserInfo T
}

type CloseEvent[T any] struct {
	UserInfo T
}

type ErrorEvent[T any] struct {
	Err      error
	UserInfo T
}

type SubscriptionOptions struct {
	EventName           string
	Filter              func(any) bool
	ReceiveSelfMessages bool
}

func NewSubscriptionsForEvents(eventNames ...string) []SubscriptionOptions {
	subscriptions := make([]SubscriptionOptions, len(eventNames))
	for i, eventName := range eventNames {
		subscriptions[i] = SubscriptionOptions{EventName: eventName, Filter: AlwaysTrue[any]()}
	}
	return subscriptions
}

func Cast[T any](e AnySocketEvent) (SocketEvent[T], error) {
	if t, ok := e.Data.(T); ok {
		return SocketEvent[T]{
			Name:   e.Name,
			Data:   t,
			Socket: e.Socket,
		}, nil
	} else if j, ok := e.Data.(json.RawMessage); ok {
		var d T
		err := json.Unmarshal(j, &d)
		if err != nil {
			return SocketEvent[T]{}, err
		}
		return SocketEvent[T]{
			Name:   e.Name,
			Data:   d,
			Socket: e.Socket,
		}, nil
	} else {
		return SocketEvent[T]{}, ErrCannotCastType
	}
}

func WithCast[T any, U any](a AnySocketEvent, callback func(T) U) U {
	event, err := Cast[T](a)
	if err != nil {
		// couldn't cast
		var zero U
		return zero
	}
	return callback(event.Data)
}

func WhenCast[T any](a AnySocketEvent, callback func(T)) {
	WithCast(a, func(t T) struct{} {
		callback(t)
		return struct{}{}
	})
}
