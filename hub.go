package websocket

import (
	"errors"
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

type HubRegistrationOptions[T any] struct {
	Subscriptions []SubscriptionOptions // Subscriptions are the subscriptions to _other events from sockets in the Hub. Events originating outside the hub using publish are unaffected. It is the responsibility of the publisher to decide what messages go where.
	UserInfo      T
}

func NewRegistrationOptionsForEvents[T any](eventNames ...string) HubRegistrationOptions[T] {
	return HubRegistrationOptions[T]{Subscriptions: NewSubscriptionsForEvents(eventNames...)}
}

func NewRegistrationOptionsWithUserInfo[T any](userInfo T) HubRegistrationOptions[T] {
	return HubRegistrationOptions[T]{UserInfo: userInfo}
}

type Hub[T any] struct {
	sockets              map[Socket]hubSocketData[T]
	requests             ro.Subject[hubRequest[T]]
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
		requests:             ro.NewSubject[hubRequest[T]](),
		closeFlag:            0,
		CloseOnNoClients:     false,
		clientsHaveExisted:   false,
		CloseTimeout:         time.Minute * 10,
		lastMessageTimestamp: time.Now(),
		events:               ro.NewSubject[AnySocketEvent](),
		hubSubscription:      ro.NewSubscription(nil),
	}
	ro.ObserveOn[hubRequest[T]](DefaultBufferSize)(hub.requests).Subscribe(ro.NewObserver(
		func(request hubRequest[T]) {
			switch {
			case request.isRegister():
				data := request.mustRegister()
				hub.registerWithOptions(data.socket, data.options)
			case request.isUnregister():
				socket := request.mustUnregister()
				hub.unregisterSocket(socket)
			case request.isPublish():
				data := request.mustPublish()
				hub.publish(data.condition, data.name, data.data)
			case request.isUserInfo():
				data := request.mustUserInfo()
				hub.setUserInfo(data.socket, data.userInfo)
			case request.isClose():
				hub.closeHub()
			}
		},
		func(err error) {
			hub.events.Error(err)
			hub.Close()
		},
		func() {
			// requests completes as a result of closing, and there's no other entry point.
			// so it's safe to do nothing here.
		},
	))
	return hub
}

type Empty = struct{}
type AnyHub = Hub[Empty]

func NewAnyHub() *AnyHub {
	return NewHub[Empty]()
}

func (h *Hub[T]) Events() ro.Observable[AnySocketEvent] {
	return h.events
}

func (h *Hub[T]) ConnectEvents() ro.Observable[SocketEvent[ConnectEvent[T]]] {
	return ro.Pipe1(h.events, Listen[ConnectEvent[T]](SocketConnectEventName))
}

func (h *Hub[T]) CloseEvents() ro.Observable[SocketEvent[CloseEvent[T]]] {
	return ro.Pipe1(h.events, Listen[CloseEvent[T]](SocketCloseEventName))
}

func (h *Hub[T]) ErrorEvents() ro.Observable[SocketEvent[ErrorEvent[T]]] {
	return ro.Pipe1(h.events, Listen[ErrorEvent[T]](SocketErrorEventName))
}

// DisconnectEvents combines the Close and Error streams so clients can
// register a single callback to handle when streams close.
func (h *Hub[T]) DisconnectEvents() ro.Observable[SocketEvent[CloseEvent[T]]] {
	errorToClose := ro.Pipe1(h.ErrorEvents(),
		ro.Map(func(e SocketEvent[ErrorEvent[T]]) SocketEvent[CloseEvent[T]] {
			return SocketEvent[CloseEvent[T]]{
				Name:   e.Name,
				Socket: e.Socket,
				Data:   CloseEvent[T]{UserInfo: e.Data.UserInfo},
			}
		}))
	return ro.Merge(h.CloseEvents(), errorToClose)
}

// Publish sends a message to all sockets whose user info satisfies a condition.
// Publish is the main entry point for non-sockets to send events to registered sockets.
// For sockets to communicate with sockets, add SubscriptionOptions when registering.
func (h *Hub[T]) Publish(condition func(T) bool, eventName string, data any) {
	h.requests.Next(newPublishRequest(condition, eventName, data))
}

// Broadcasts publishes a message to all sockets unconditionally.
func (h *Hub[T]) Broadcast(eventName string, data any) {
	h.Publish(AlwaysTrue[T](), eventName, data)
}

// Register registers a client to receive messages.
// Blocks until the client is registered.
func (h *Hub[T]) Register(socket Socket) {
	h.RegisterWithOptions(socket, HubRegistrationOptions[T]{})
}

// Register registers a client with the given options to receive messages.
// Blocks until the client is registered.
func (h *Hub[T]) RegisterWithOptions(socket Socket, options HubRegistrationOptions[T]) {
	h.requests.Next(newRegisterRequest[T](socket, options))
}

// unregisterSocket removes a client from the hub. This is an internal method
// used only by the hub itself to manage socket lifecycle.
func (h *Hub[T]) UnegisterSocket(socket Socket) {
	h.requests.Next(newUnregisterRequest[T](socket))
}

// Close closes the hub and all connected sockets.
func (h *Hub[T]) Close() {
	h.requests.Next(newCloseRequest[T]())
}

func (h *Hub[T]) GetUserInfo(socket Socket) (T, error) {
	if u, ok := h.sockets[socket]; ok {
		return u.userInfo, nil
	}
	var output T
	return output, ErrSocketNotFound
}

func (h *Hub[T]) SetUserInfo(socket Socket, userInfo T) {
	h.requests.Next(newUserInfoRequest(socket, userInfo))
}

func (h *Hub[T]) registerWithOptions(socket Socket, options HubRegistrationOptions[T]) {
	if _, ok := h.sockets[socket]; ok {
		h.events.Next(AnySocketEvent{SocketErrorEventName, ErrSocketAlreadyRegistered, socket})
		h.emitError(ErrSocketAlreadyRegistered)
		return
	}
	h.clientsHaveExisted = true
	subscription := socket.Events().Subscribe(ro.NewObserver(
		func(event AnySocketEvent) {
			h.events.Next(event)
		},
		func(err error) {
			data, _ := h.sockets[socket]
			delete(h.sockets, socket)
			h.events.Next(AnySocketEvent{
				SocketErrorEventName,
				ErrorEvent[T]{err, data.userInfo},
				socket,
			})
		},
		func() {
			data, _ := h.sockets[socket]
			delete(h.sockets, socket)
			h.events.Next(AnySocketEvent{
				SocketCloseEventName,
				CloseEvent[T]{data.userInfo},
				socket,
			})
		},
	))
	h.hubSubscription.AddUnsubscribable(subscription)
	data := hubSocketData[T]{socket: socket, userInfo: options.UserInfo}
	h.sockets[socket] = data
	for _, option := range options.Subscriptions {
		h.subscribeSocket(socket, option)
	}
	h.events.Next(AnySocketEvent{SocketConnectEventName, ConnectEvent[T]{data.userInfo}, socket})
}

func (h *Hub[T]) unregisterSocket(socket Socket) {
	// Check if socket is still registered to prevent double-unregistration
	socketData, ok := h.sockets[socket]
	if ok {
		// the close completion handler is responsible for deleting it from the map.
		// Basically, unregister requests a close, the close handler completes it.
		socketData.socket.Close()
	} else {
		h.events.Next(AnySocketEvent{SocketErrorEventName, ErrSocketNotRegistered, socket})
		h.emitError(ErrSocketNotRegistered)
		return
	}
	h.postRemoveSocketHook()
}

func (h *Hub[T]) publish(condition func(T) bool, eventName string, data any) {
	for socket, hubData := range h.sockets {
		if condition(hubData.userInfo) {
			socket.Send(AnySocketEvent{Name: eventName, Data: data})
		}
	}
}

func (h *Hub[T]) setUserInfo(socket Socket, userInfo T) {
	socketData, ok := h.sockets[socket]
	if !ok {
		h.events.Next(AnySocketEvent{SocketErrorEventName, ErrSocketNotRegistered, socket})
		return
	}
	socketData.userInfo = userInfo
	h.sockets[socket] = socketData
}

func (h *Hub[T]) closeHub() {
	h.requests.Complete()
	h.closeAllSockets()
	// events must complete after closing sockets or else they won't
	// have a chance to listen to their complete events.
	h.events.Complete()
	h.hubSubscription.Unsubscribe()
}

func (h *Hub[T]) closeAllSockets() {
	for _, s := range h.sockets {
		s.socket.Close()
	}
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
