package websocket

// Client is an object that listens for messages
// from a Hub. A Client may only be reigstered with
// one Hub at a time.
type Client interface {
	// Send is the channel receiving messages fo this client.
	// Clients should almost always read from this channel
	// on a background goroutine.
	Send() chan<- ClientEvent

	// Closes the client. May block until the client is closed.
	Close()
}

// emptyClient is a dummy client used by Hub to allow broadcasting
// messages not originating from a client. Clients consume messages _and_
// produce messages, but not all producers consume messages.
type emptyClient struct {
	send chan ClientEvent
}

func (e *emptyClient) Send() chan<- ClientEvent {
	return e.send
}

func (e *emptyClient) Close() {
	close(e.send)
}
