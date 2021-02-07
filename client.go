package websocket

// Client is an object that listens for messages
// from a Hub. A Client may only be reigstered with
// one Hub at a time.
type Client interface {
	// Send is the channel receiving messages fo this client.
	Send() chan<- ClientEvent

	// Closes the client. May block unitl the client is closed.
	Close()
}
