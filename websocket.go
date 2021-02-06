package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Generate socket.js.go.
//go:generate tsc --lib dom,es2015 test/socket.ts
//go:generate go run generate/generate.go

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ServeWebsocket upgrades an HTTP request to a websocket connection.
//   hub is the Hub to register the client with.
//   w is the ResponseWriter associated with the request.
//   req is the Request.
//   onClose is a function to run when the client is disconnected from the Hub.
func ServeWebsocket(hub *Hub, w http.ResponseWriter, req *http.Request, onClose func(*Hub)) error {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return err
	}
	client := WebsocketClient{hub: hub, conn: conn, send: make(chan Event, 256)}
	client.hub.Register(&client, ClientRegistrationOptions{false, onClose})

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
	return nil
}

// ServeSocketJs serves a Javascript file containing a Socket class
// which abstracts a native WebSocket element. It allows the sending
// of named events instead of raw strings, conforming to Hub's API.
func ServeSocketJs(w http.ResponseWriter, req *http.Request) error {
	_, err := w.Write([]byte(socketJsContents))
	return err
}
