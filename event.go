package websocket

import (
	"encoding/json"
)

// A type-erased event sent to or from a socket.
type AnyEvent struct {
	Name string `json:"name"`
	Data any    `json:"any"`
}

// SocketEvent represents an event with a known provenance.
// Socket may be nil, but that represents an event with no
// source, such as a programatically determined one.
type SocketEvent struct {
	Socket Socket
	Event  AnyEvent
}

type SocketErrorEvent struct {
	Socket Socket
	Err    error
}

type JSONEvent struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

// Event is a type-erased event sent over a socket.
type Event2 struct {
	// EventName is the name of the event. Both senders and receivers are responsible
	// for using the correct event name.
	EventName string `json:"eventName"`
	// Data is the raw serialized event data.
	Data json.RawMessage `json:"data"`
}
