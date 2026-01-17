package websocket

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/samber/ro"
)

var (
	ErrSocketAlreadyRegistered = errors.New("socket already registered")
	ErrSocketNotRegistered     = errors.New("socket not registered")
)

type hubSocketData struct {
	socket       Socket
	userInfo     UserInfo
	subscription ro.Subscription
}

type registerData struct {
	socket   Socket
	userInfo UserInfo
}

type Hub struct {
	sockets              map[Socket]hubSocketData
	register             chan registerData
	unregister           chan Socket
	close                chan bool
	closeFlag            int32
	CloseOnNoClients     bool
	clientsHaveExisted   bool
	CloseTimeout         time.Duration
	subscriptions        []ro.Subscription
	lastMessageTimestamp time.Time
	events               ro.Subject[AnySocketEvent]
	hubSubscription      ro.Subscription // hubSubscription is a parent subscription for all sockets attached to this hub, for easy closing.
}

// Creates and runs a Hub.
func NewHub() *Hub {
	hub := &Hub{
		sockets:              make(map[Socket]hubSocketData),
		register:             make(chan registerData),
		unregister:           make(chan Socket),
		close:                make(chan bool),
		closeFlag:            0,
		CloseOnNoClients:     false,
		clientsHaveExisted:   false,
		CloseTimeout:         time.Minute * 10,
		subscriptions:        []ro.Subscription{},
		lastMessageTimestamp: time.Now(),
		events:               ro.NewSubject[AnySocketEvent](),
		hubSubscription:      ro.NewSubscription(nil),
	}
	go hub.Run()
	return hub
}

// Run listens for register, unregister, broadcast, and close events.
// Blocks while the hub is running. Run on a separate goroutine
// if you do not wish to block.

func (h *Hub) Run() {
	defer h.closeHub()
	defer h.events.Complete()
	defer close(h.unregister)
	defer close(h.register)
	timeoutTicker := time.NewTicker(h.CloseTimeout)
	defer timeoutTicker.Stop()
	for {
		select {
		case data := <-h.register:
			if _, ok := h.sockets[data.socket]; ok {
				h.events.Next(AnySocketEvent{SocketErrorEventName, ErrSocketAlreadyRegistered, data.socket, data.userInfo})
				h.emitError(ErrSocketAlreadyRegistered)
				h.Close()
				continue
			}
			h.clientsHaveExisted = true
			socket := data.socket
			subscription := data.socket.Events().Subscribe(ro.NewObserver(
				func(event AnyEvent) {
					h.events.Next(AnySocketEvent{event.Name, event.Data, data.socket, data.userInfo})
				},
				func(err error) {
					delete(h.sockets, socket)
					h.events.Next(AnySocketEvent{SocketErrorEventName, err, data.socket, data.userInfo})
					h.postRemoveSocketHook()
				},
				func() {
					delete(h.sockets, socket)
					h.events.Next(AnySocketEvent{SocketCloseEventName, nil, data.socket, data.userInfo})
					h.postRemoveSocketHook()
				},
			))
			h.hubSubscription.AddUnsubscribable(subscription)
			h.sockets[socket] = hubSocketData{socket, data.userInfo, subscription}
			h.events.Next(AnySocketEvent{SocketConnectEventName, nil, socket, data.userInfo})
		case socket := <-h.unregister:
			socketData, ok := h.sockets[socket]
			if ok {
				h.closeSocket(socketData)
			} else {
				h.events.Next(AnySocketEvent{SocketErrorEventName, ErrSocketNotRegistered, socket, socketData.userInfo})
				h.emitError(ErrSocketNotRegistered)
				h.Close()
				continue
			}
			h.postRemoveSocketHook()
		case _ = <-h.close:
			// This only occurs when Close() has been called, guaranteeing that the
			// closeFlag is always set before closing.
			//
			// the defer call ensures all sockets get closed.
			return
		case _ = <-timeoutTicker.C:
			if time.Since(h.lastMessageTimestamp) >= h.CloseTimeout {
				h.Close()
			}
			timeoutTicker.Stop()
			timeoutTicker = time.NewTicker(h.CloseTimeout)
		}
	}
}

func (h *Hub) Events() ro.Observable[AnySocketEvent] {
	return h.events
}

// Register registers a client with the given options to receive messages.
// Blocks until the client is registered.
func (h *Hub) Register(socket Socket, userInfo UserInfo) {
	h.register <- registerData{socket, userInfo}
}

// Unregister removes a client. Blocks until the client is unregistered.
func (h *Hub) Unregister(socket Socket) {
	h.unregister <- socket
}

func (h *Hub) Close() {
	if atomic.CompareAndSwapInt32(&h.closeFlag, 0, 1) {
		// Must be called on a separate goroutine, because if this occurs due to
		// a Close event or an unregister event, this will execute on the Run goroutine,
		// preventing it from ever unblocking the close event.
		go func() { h.close <- true }()
	}
}

func (h *Hub) closeHub() {
	h.closeAllSockets()
	h.hubSubscription.Unsubscribe()
}

func (h *Hub) closeAllSockets() {
	for _, s := range h.sockets {
		h.closeSocket(s)
	}
}

func (h *Hub) closeSocket(socket hubSocketData) {
	delete(h.sockets, socket.socket)
	socket.socket.Close()
	// no need to unsubscribe because hubSubscription owns all unsubscribing.
}

func (h *Hub) postRemoveSocketHook() {
	if len(h.sockets) == 0 && h.CloseOnNoClients && h.clientsHaveExisted {
		// Use Close, which will delay until another loop can read from h.close,
		// to ensure the atomic closeFlag is always set before closing.
		h.Close()
	}
}

func (h *Hub) emitError(err error) {
	h.events.Error(err)
}
