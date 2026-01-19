package websocket

import "github.com/samber/mo"

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

type _hubRequest[T any] = mo.Either5[hubRegistrationData[T], Socket, publishData[T], hubUserInfoData[T], bool]
type hubRequest[T any] _hubRequest[T]

func newRegisterRequest[T any](socket Socket, options HubRegistrationOptions[T]) hubRequest[T] {
	return hubRequest[T](mo.NewEither5Arg1[hubRegistrationData[T], Socket, publishData[T], hubUserInfoData[T], bool](hubRegistrationData[T]{socket: socket, options: options}))
}

func newUnregisterRequest[T any](socket Socket) hubRequest[T] {
	return hubRequest[T](mo.NewEither5Arg2[hubRegistrationData[T], Socket, publishData[T], hubUserInfoData[T], bool](socket))
}

func newPublishRequest[T any](condition func(T) bool, name string, data any) hubRequest[T] {
	return hubRequest[T](mo.NewEither5Arg3[hubRegistrationData[T], Socket, publishData[T], hubUserInfoData[T], bool](publishData[T]{condition: condition, name: name, data: data}))
}

func newUserInfoRequest[T any](socket Socket, userInfo T) hubRequest[T] {
	return hubRequest[T](mo.NewEither5Arg4[hubRegistrationData[T], Socket, publishData[T], hubUserInfoData[T], bool](hubUserInfoData[T]{socket: socket, userInfo: userInfo}))
}

func newCloseRequest[T any]() hubRequest[T] {
	return hubRequest[T](mo.NewEither5Arg5[hubRegistrationData[T], Socket, publishData[T], hubUserInfoData[T], bool](true))
}

func (h hubRequest[T]) isRegister() bool {
	return (_hubRequest[T])(h).IsArg1()
}

func (h hubRequest[T]) mustRegister() hubRegistrationData[T] {
	return (_hubRequest[T])(h).MustArg1()
}

func (h hubRequest[T]) isUnregister() bool {
	return (_hubRequest[T])(h).IsArg2()
}

func (h hubRequest[T]) mustUnregister() Socket {
	return (_hubRequest[T])(h).MustArg2()
}

func (h hubRequest[T]) isPublish() bool {
	return (_hubRequest[T])(h).IsArg3()
}

func (h hubRequest[T]) mustPublish() publishData[T] {
	return (_hubRequest[T])(h).MustArg3()
}

func (h hubRequest[T]) isUserInfo() bool {
	return (_hubRequest[T])(h).IsArg4()
}

func (h hubRequest[T]) mustUserInfo() hubUserInfoData[T] {
	return (_hubRequest[T])(h).MustArg4()
}

func (h hubRequest[T]) isClose() bool {
	return (_hubRequest[T])(h).IsArg5()
}
