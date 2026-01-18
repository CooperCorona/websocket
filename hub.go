package websocket

import (
	"errors"
	"fmt"
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
	socket   Socket
	userInfo T
}

type HubRegistrationOptions struct {
	Subscriptions []SubscriptionOptions // Subscriptions are the subscriptions to _other events from sockets in the Hub. Events originating outside the hub using publish are unaffected. It is the responsibility of the publisher to decide what messages go where.
}

func NewRegistrationOptionsForEvents(eventNames ...string) HubRegistrationOptions {
	return HubRegistrationOptions{Subscriptions: NewSubscriptionsForEvents(eventNames...)}
}

type hubRegistrationData struct {
	socket  Socket
	options HubRegistrationOptions
}

type Hub[T any] struct {
	sockets              map[Socket]hubSocketData[T]
	register             chan hubRegistrationData
	unregister           chan Socket
	close                chan bool
	closeFlag            int32
	CloseOnNoClients     bool
	clientsHaveExisted   bool
	CloseTimeout         time.Duration
	lastMessageTimestamp time.Time
	events               ro.Subject[AnySocketEvent]
	hubSubscription      ro.Subscription // hubSubscription is a parent subscription for all sockets attached to this hub, for easy closing.
}

// Creates and runs a Hub.
func NewHub[T any]() *Hub[T] {
	hub := &Hub[T]{
		sockets:              make(map[Socket]hubSocketData[T]),
		register:             make(chan hubRegistrationData),
		unregister:           make(chan Socket),
		close:                make(chan bool),
		closeFlag:            0,
		CloseOnNoClients:     false,
		clientsHaveExisted:   false,
		CloseTimeout:         time.Minute * 10,
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
		case data := <-h.register:
			socket := data.socket
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
			h.sockets[socket] = hubSocketData[T]{socket: socket}
			for _, option := range data.options.Subscriptions {
				h.subscribeSocket(socket, option)
			}
			h.events.Next(AnySocketEvent{SocketConnectEventName, nil, socket})
			fmt.Printf("Registered socket\n")
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

// Publish sends a message to all sockets whose user info satisfies a condition.
// Publish is the main entry point for non-sockets to send events to registered sockets.
// For sockets to communicate with sockets, add SubscriptionOptions when registering.
func (h *Hub[T]) Publish(condition func(T) bool, eventName string, data any) {
	for socket, hubData := range h.sockets {
		if condition(hubData.userInfo) {
			socket.Send(AnySocketEvent{Name: eventName, Data: data})
		}
	}
}

// Broadcasts publishes a message to all sockets unconditionally.
func (h *Hub[T]) Broadcast(eventName string, data any) {
	h.Publish(AlwaysTrue[T](), eventName, data)
}

// Register registers a client to receive messages.
// Blocks until the client is registered.
func (h *Hub[T]) Register(socket Socket) {
	h.RegisterWithOptions(socket, HubRegistrationOptions{})
}

// Register registers a client with the given options to receive messages.
// Blocks until the client is registered.
func (h *Hub[T]) RegisterWithOptions(socket Socket, options HubRegistrationOptions) {
	h.register <- hubRegistrationData{socket, options}
	fmt.Printf("Register unblocked\n")
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

// this doesn't work, because I'm manually sending events in Publish, not
// using an observable. Furthermore, where is the callback?
// Maybe stub needs to be able to attach its own callback to individual events.
// After all, a Websocket doesn't have callbacks. "Send" is a method in the interface,
// not something that needs to be abstract. It's already abstract by definition of
// the interface. The stub needs to be abstract to support stubbing.
func (h *Hub[T]) subscribeSocket(socket Socket, options SubscriptionOptions) {
	sub := ro.Pipe3(h.Events(),
		ro.Filter(func(e AnySocketEvent) bool {
			return e.Socket != socket || options.ReceiveSelfMessages
		}),
		ListenAny(options.EventName),
		ro.Filter(func(e AnySocketEvent) bool {
			return options.Filter(e.Data)
		}),
	).Subscribe(ro.OnNext(func(e AnySocketEvent) {
		socket.Send(e)
	}))
	h.hubSubscription.AddUnsubscribable(sub)
}

func (h *Hub[T]) emitError(err error) {
	h.events.Error(err)
}
