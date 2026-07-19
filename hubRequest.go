package websocket

type hubRequestKind int

const (
	hubRequestRegister hubRequestKind = iota
	hubRequestUnregister
	hubRequestPublish
	hubRequestUserInfo
	hubRequestClose
	hubRequestSocketEvent  // an event was received from a registered socket
	hubRequestSocketError  // a registered socket's stream errored
	hubRequestSocketClosed // a registered socket's stream completed
)

type hubRegistrationData[T any] struct {
	socket  Socket
	options HubRegistrationOptions[T]
}

type hubUserInfoData[T any] struct {
	socket   Socket
	userInfo T
}

type publishData[T any] struct {
	condition func(T) bool
	name      string
	data      any
}

type hubRequest[T any] struct {
	kind         hubRequestKind
	registration hubRegistrationData[T]
	socket       Socket
	publish      publishData[T]
	userInfo     hubUserInfoData[T]
	event        AnyEvent
	err          error
}

func newRegisterRequest[T any](socket Socket, options HubRegistrationOptions[T]) hubRequest[T] {
	return hubRequest[T]{kind: hubRequestRegister, registration: hubRegistrationData[T]{socket: socket, options: options}}
}

func newUnregisterRequest[T any](socket Socket) hubRequest[T] {
	return hubRequest[T]{kind: hubRequestUnregister, socket: socket}
}

func newPublishRequest[T any](condition func(T) bool, name string, data any) hubRequest[T] {
	return hubRequest[T]{kind: hubRequestPublish, publish: publishData[T]{condition: condition, name: name, data: data}}
}

func newUserInfoRequest[T any](socket Socket, userInfo T) hubRequest[T] {
	return hubRequest[T]{kind: hubRequestUserInfo, userInfo: hubUserInfoData[T]{socket: socket, userInfo: userInfo}}
}

func newCloseRequest[T any]() hubRequest[T] {
	return hubRequest[T]{kind: hubRequestClose}
}

func newSocketEventRequest[T any](socket Socket, event AnyEvent) hubRequest[T] {
	return hubRequest[T]{kind: hubRequestSocketEvent, socket: socket, event: event}
}

func newSocketErrorRequest[T any](socket Socket, err error) hubRequest[T] {
	return hubRequest[T]{kind: hubRequestSocketError, socket: socket, err: err}
}

func newSocketClosedRequest[T any](socket Socket) hubRequest[T] {
	return hubRequest[T]{kind: hubRequestSocketClosed, socket: socket}
}
