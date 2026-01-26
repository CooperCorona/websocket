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

// A type-erased event structed as JSON so websockets don't parse into maps,
// which cannot be gracefully parsed to an explicit type later.
type JSONEvent struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

// AnySocketEvent represents a type-erased event with a known provenance.
// Socket may be nil, but that represents an event with no
// source, such as a programatically determined one.
type AnySocketEvent[U any] struct {
	Name     string
	Data     any
	Socket   Socket
	UserInfo U
}

func AsJSON(a any) (json.RawMessage, error) {
	if j, ok := a.(json.RawMessage); ok {
		return j, nil
	}
	return json.Marshal(a)
}

func (a AnySocketEvent[U]) AsAnyEvent() (AnyEvent, error) {
	return AnyEvent{Name: a.Name, Data: a.Data}, nil
}

// SocketEvent is a parameterized event with a known provenance.
// Socket may be nil, but that represents an event with no
// source, such as a programatically determined one.
type SocketEvent[T any, U any] struct {
	Name     string
	Data     T
	Socket   Socket
	UserInfo U
}
type EmptySocketEvent[U any] = SocketEvent[Empty, U]

func (a SocketEvent[T, U]) AsAnyEvent() (AnyEvent, error) {
	return AnySocketEvent[U]{Name: a.Name, Data: a.Data, Socket: a.Socket}.AsAnyEvent()
}

type ErrorEvent struct {
	Err error
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

func Cast[T any, U any](e AnySocketEvent[U]) (SocketEvent[T, U], error) {
	if t, ok := e.Data.(T); ok {
		return SocketEvent[T, U]{
			Name:     e.Name,
			Data:     t,
			Socket:   e.Socket,
			UserInfo: e.UserInfo,
		}, nil
	} else if j, ok := e.Data.(json.RawMessage); ok {
		var d T
		err := json.Unmarshal(j, &d)
		if err != nil {
			return SocketEvent[T, U]{}, err
		}
		return SocketEvent[T, U]{
			Name:     e.Name,
			Data:     d,
			Socket:   e.Socket,
			UserInfo: e.UserInfo,
		}, nil
	} else {
		return SocketEvent[T, U]{}, ErrCannotCastType
	}
}

func WithCast[T any, U any, V any](a AnySocketEvent[U], callback func(SocketEvent[T, U]) V) V {
	event, err := Cast[T](a)
	if err != nil {
		// couldn't cast
		var zero V
		return zero
	}
	return callback(event)
}

func WhenCast[T any, U any](a AnySocketEvent[U], callback func(SocketEvent[T, U])) {
	WithCast(a, func(t SocketEvent[T, U]) struct{} {
		callback(t)
		return struct{}{}
	})
}

func On[T any, U any](eventName string, a AnySocketEvent[U], callback func(SocketEvent[T, U])) {
	if a.Name != eventName {
		return
	}
	WhenCast(a, callback)
}
