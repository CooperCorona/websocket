package websocket

import (
	"encoding/json"
)

type AnyEvent struct {
	Name string `json:"name"`
	Data any    `json:"any"`
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
