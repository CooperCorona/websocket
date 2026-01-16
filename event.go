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

// SocketEvent represents an event with a known provenance.
// Socket may be nil, but that represents an event with no
// source, such as a programatically determined one.
type SocketEvent struct {
	Name   string
	Data   any
	Socket Socket
	// Arbitrary data stored alongside the socket.
	UserInfo UserInfo
}
