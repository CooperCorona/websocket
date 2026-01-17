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
	ErrSocketNotFound          = errors.New("socket not found")
)

type hubSocketData[T any] struct {
	socket       Socket
	subscription ro.Subscription
	userInfo     T
}

type Hub[T any] struct {
	sockets              map[Socket]hubSocketData[T]
	register             chan Socket
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
func NewHub[T any]() *Hub[T] {
	hub := &Hub[T]{
		sockets:              make(map[Socket]hubSocketData[T]),
		register:             make(chan Socket),
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

type Empty = struct{}
type AnyHub = Hub[Empty]

func NewAnyHub() *AnyHub {
	return NewHub[Empty]()
}

// Run listens for register, unregister, broadcast, and close events.
// Blocks while the hub is running. Run on a separate goroutine
// if you do not wish to block.

func (h *Hub[T]) Run() {
	defer h.closeHub()
	defer close(h.unregister)
	defer close(h.register)
	timeoutTicker := time.NewTicker(h.CloseTimeout)
	defer timeoutTicker.Stop()
	for {
		select {
		case socket := <-h.register:
			if _, ok := h.sockets[socket]; ok {
				h.events.Next(AnySocketEvent{SocketErrorEventName, ErrSocketAlreadyRegistered, socket})
				h.emitError(ErrSocketAlreadyRegistered)
				h.Close()
				continue
			}
			h.clientsHaveExisted = true
			subscription := socket.Events().Subscribe(ro.NewObserver(
				func(event AnySocketEvent) {
					h.events.Next(event)
				},
				func(err error) {
					delete(h.sockets, socket)
					h.events.Next(AnySocketEvent{SocketErrorEventName, err, socket})
					h.postRemoveSocketHook()
				},
				func() {
					delete(h.sockets, socket)
					h.events.Next(AnySocketEvent{SocketCloseEventName, nil, socket})
					h.postRemoveSocketHook()
				},
			))
			h.hubSubscription.AddUnsubscribable(subscription)
			h.sockets[socket] = hubSocketData[T]{socket: socket, subscription: subscription}
			h.events.Next(AnySocketEvent{SocketConnectEventName, nil, socket})
		case socket := <-h.unregister:
			socketData, ok := h.sockets[socket]
			if ok {
				h.closeSocket(socketData)
			} else {
				h.events.Next(AnySocketEvent{SocketErrorEventName, ErrSocketNotRegistered, socket})
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

func (h *Hub[T]) Events() ro.Observable[AnySocketEvent] {
	return h.events
}

// Register registers a client with the given options to receive messages.
// Blocks until the client is registered.
func (h *Hub[T]) Register(socket Socket) {
	h.register <- socket
}

// Unregister removes a client. Blocks until the client is unregistered.
func (h *Hub[T]) Unregister(socket Socket) {
	h.unregister <- socket
}

func (h *Hub[T]) Close() {
	if atomic.CompareAndSwapInt32(&h.closeFlag, 0, 1) {
		// Must be called on a separate goroutine, because if this occurs due to
		// a Close event or an unregister event, this will execute on the Run goroutine,
		// preventing it from ever unblocking the close event.
		go func() { h.close <- true }()
	}
}

func (h *Hub[T]) GetUserInfo(socket Socket) (T, error) {
	if u, ok := h.sockets[socket]; ok {
		return u.userInfo, nil
	}
	var output T
	return output, ErrSocketNotFound
}

func (h *Hub[T]) SetUserInfo(socket Socket, userInfo T) error {
	if u, ok := h.sockets[socket]; ok {
		u.userInfo = userInfo
		h.sockets[socket] = u
		return nil
	}
	return ErrSocketNotFound
}

func (h *Hub[T]) closeHub() {
	h.events.Complete()
	h.closeAllSockets()
	h.hubSubscription.Unsubscribe()
}

func (h *Hub[T]) closeAllSockets() {
	for _, s := range h.sockets {
		h.closeSocket(s)
	}
}

func (h *Hub[T]) closeSocket(socket hubSocketData[T]) {
	delete(h.sockets, socket.socket)
	socket.socket.Close()
	// no need to unsubscribe because hubSubscription owns all unsubscribing.
}

func (h *Hub[T]) postRemoveSocketHook() {
	if len(h.sockets) == 0 && h.CloseOnNoClients && h.clientsHaveExisted {
		// Use Close, which will delay until another loop can read from h.close,
		// to ensure the atomic closeFlag is always set before closing.
		h.Close()
	}
}

func (h *Hub[T]) emitError(err error) {
	h.events.Error(err)
}
