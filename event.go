package websocket

const (
	SocketConnectEventName = "SocketConnectEvent"
	SocketCloseEventName   = "SocketCloseEventName"
	SocketErrorEventName   = "SocketErrorEventName"
)

type UserInfo = map[string]any

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
	// Arbitrary data stored alongside the socket.
	UserInfo UserInfo
}

// SocketEvent is a parameterized event with a known provenance.
// Socket may be nil, but that represents an event with no
// source, such as a programatically determined one.
type SocketEvent[T any] struct {
	Name   string
	Data   T
	Socket Socket
	// Arbitrary data stored alongside the socket.
	UserInfo UserInfo
}
