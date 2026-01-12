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

/*
	type Hub2 struct {
		sockets            []Socket
		broadcast          chan ClientEvent
		register           chan clientData
		unregister         chan Client
		clients            map[Client]clientData
		close              chan bool
		closeFlag          int
		CloseOnNoClients   bool
		clientsHaveExisted bool
		CloseTimeout       time.Duration
	}

	func NewHub2() *Hub2 {
		return &Hub2{
			sockets:            []Socket{},
			broadcast:          make(chan ClientEvent),
			register:           make(chan clientData),
			unregister:         make(chan Client),
			clients:            make(map[Client]clientData),
			close:              make(chan bool),
			closeFlag:          0,
			CloseOnNoClients:   false,
			clientsHaveExisted: false,
			CloseTimeout:       time.Minute * 10}
	}

// Run listens for register, unregister, broadcast, and close events.
// Blocks while the hub is running. Run on a separate goroutine
// if you do not wish to block.

	func (h *Hub2) Run() {
		defer h.closeAllClients()
		timeoutTicker := time.NewTicker(h.CloseTimeout)
		defer timeoutTicker.Stop()
		for {
			select {
			case clientData := <-h.register:
				h.clients[clientData.client] = clientData
				h.clientsHaveExisted = true
			case client := <-h.unregister:
				if clientData, ok := h.clients[client]; ok {
					h.closeClient(client, clientData)
				}
				if len(h.clients) == 0 && h.CloseOnNoClients && h.clientsHaveExisted {
					// Use Close, which will delay until another loop can read from h.close,
					// to ensure the atomic closeFlag is always set before closing.
					h.Close()
				}
			case clientEvent := <-h.broadcast:
				h.lastMessageTimestamp = time.Now()
				for client, clientData := range h.clients {
					if !clientData.receiveSelfMessages && clientEvent.Client == client {
						// This message was sent by the current client, but the current
						// client does not receive its own messages. Skip it.
						continue
					}
					select {
					case client.Send() <- clientEvent:
					default:
						h.closeClient(client, clientData)
					}
				}
				if len(h.clients) == 0 && h.CloseOnNoClients && h.clientsHaveExisted {
					h.Close()
				}
			case _ = <-h.close:
				// This only occurs when Close() has been called, guaranteeing that the
				// closeFlag is always set before closing.
				return
			case _ = <-timeoutTicker.C:
				if time.Now().Sub(h.lastMessageTimestamp) >= h.CloseTimeout {
					h.Close()
				}
				timeoutTicker.Stop()
				timeoutTicker = time.NewTicker(h.CloseTimeout)
			}
		}
	}
*/
type Socket interface {
	Send(AnyEvent)

	Events() ro.Observable[AnyEvent]

	// Closes the client. May block until the client is closed.
	Close()
}

// we still need a Socket to represent input AND output. Maybe we will want a Hub to
// abstract broadcasting. But we can dispense with the channel and just use a Subject[AnyEvent].
// Then use this custom function to filter for event names and get the strongly typed version
// in response.
func Listen[T any](eventName string) func(ro.Observable[AnyEvent]) ro.Observable[T] {
	return func(input ro.Observable[AnyEvent]) ro.Observable[T] {
		return ro.Pipe3(input,
			ro.Map(func(e AnyEvent) *T {
				if e.Name != eventName {
					return nil
				}
				if t, ok := e.Data.(T); ok {
					return &t
				} else if j, ok := e.Data.(json.RawMessage); ok {
					var d T
					err := json.Unmarshal(j, &d)
					if err != nil {
						return nil
					}
					return &d
				} else {
					// must be some other type. No way to know if the type was intentional or not,
					// so we return nil and stop processing.
					return nil
				}
			}),
			ro.Filter(func(t *T) bool {
				return t != nil
			}),
			ro.Map(func(t *T) T {
				// t is guaranteed to be non-nil by the time we reach here.
				return *t
			}),
		)
	}
}

// SocketStub is a Socket you can manually send events to.
type SocketStub struct {
	subject     ro.Subject[AnyEvent]
	subscribers map[ro.Subscription]struct{}
}

type ConfigurationOptions struct {
	BufferSize int
}

func NewStub(options ConfigurationOptions) *SocketStub {
	return &SocketStub{ro.NewSubject[AnyEvent](), make(map[ro.Subscription]struct{})}
}

func (s *SocketStub) Send(event AnyEvent) {
	s.subject.Next(event)
}

func (s *SocketStub) Events() ro.Observable[AnyEvent] {
	return s.subject
}

func (s *SocketStub) AddSubscriber(subscription ro.Subscription) {
	s.subscribers[subscription] = struct{}{}
}

func (s *SocketStub) RemoveSubscriber(subscription ro.Subscription) {
	delete(s.subscribers, subscription)
}

func (s *SocketStub) Close() {
	s.subject.Complete()
	for subscription, _ := range s.subscribers {
		subscription.Unsubscribe()
	}
}

func (s *SocketStub) Post(name string, data any) {
	s.Send(AnyEvent{name, data})
}

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
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		var event AnyEvent
		err = json.Unmarshal(message, &event)
		if err != nil {
			log.Printf("error marshalling bytes: %v. Skipping message", err)
			continue
		}
		w.observable.Next(event)
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
		case clientEvent, ok := <-w.send:
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
			message, err := json.Marshal(clientEvent)
			if err == nil {
				writer.Write(message)
			} else {
				log.Printf("failed to marshal event: %v. skipping", clientEvent.Name)
			}

			if err := writer.Close(); err != nil {
				return
			}
		case <-ticker.C:
			w.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
