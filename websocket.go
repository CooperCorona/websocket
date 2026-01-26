package websocket

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/samber/ro"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// size of the buffer for Websocket channels.
	DefaultBufferSize = 8
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Websocket is a Socket sending and receiving messages from an HTTP Websocket.
type Websocket struct {
	// The websocket connection.
	conn *websocket.Conn

	send chan AnyEvent

	// observable is an observable wrapping emit.
	observable ro.Subject[AnyEvent]
}

func NewWebsocket(conn *websocket.Conn, options ConfigurationOptions) Websocket {
	send := make(chan AnyEvent, options.BufferSize)
	observable := ro.NewSubject[AnyEvent]()
	return Websocket{conn, send, observable}
}

var DefaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func UpgradeWebsocket(upgrader websocket.Upgrader, w http.ResponseWriter, req *http.Request) (*Websocket, error) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return nil, err
	}
	client := NewWebsocket(conn, ConfigurationOptions{BufferSize: DefaultBufferSize})

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
	return &client, nil
}

func (w *Websocket) Send(event AnyEvent) {
	w.send <- event
}

func (w *Websocket) Events() ro.Observable[AnyEvent] {
	return w.observable
}

func (w *Websocket) Close() {
	close(w.send)
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (w *Websocket) readPump() {
	defer func() {
		w.conn.Close()
	}()
	w.conn.SetReadLimit(maxMessageSize)
	w.conn.SetReadDeadline(time.Now().Add(pongWait))
	w.conn.SetPongHandler(func(string) error { w.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := w.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("error: %v", err)
				w.observable.Error(err)
			} else {
				w.observable.Complete()
			}
			break
		}
		message = bytes.TrimSpace(bytes.ReplaceAll(message, newline, space))
		var event JSONEvent
		err = json.Unmarshal(message, &event)
		if err != nil {
			log.Printf("error marshalling bytes: %v. Skipping message", err)
			continue
		}
		w.observable.Next(AnyEvent{event.Name, event.Data})
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (w *Websocket) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		w.conn.Close()
	}()
	for {
		select {
		case socketEvent, ok := <-w.send:
			w.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				w.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			writer, err := w.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			// because AnyEvent isn't JSON, we must parse it into
			// bytes before we can serialize the entire message.
			dataBytes, err := json.Marshal(socketEvent.Data)
			if err == nil {
				event := AnyEvent{Name: socketEvent.Name, Data: dataBytes}
				message, err := json.Marshal(event)
				if err == nil {
					writer.Write(message)
				} else {
					log.Printf("failed to marshal event: %v. skipping", socketEvent.Name)
				}

				if err := writer.Close(); err != nil {
					return
				}
			} else {
				log.Printf("failed to marshal event: %v due to %v. skipping", socketEvent.Name, err)
			}
		case <-ticker.C:
			w.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
