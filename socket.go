package websocket

import (
	"github.com/samber/ro"
)

type Socket interface {
	Send(AnySocketEvent)

	Events() ro.Observable[AnySocketEvent]

	// Closes the client. May block until the client is closed.
	Close()
}

// SocketStub is a Socket you can manually send events to.
type SocketStub struct {
	subject    ro.Subject[AnySocketEvent]
	sentEvents ro.Subject[AnySocketEvent]
}

type ConfigurationOptions struct {
	BufferSize int
}

func NewStub(options ConfigurationOptions) *SocketStub {
	return &SocketStub{ro.NewSubject[AnySocketEvent](), ro.NewSubject[AnySocketEvent]()}
}

func (s *SocketStub) Send(event AnySocketEvent) {
	s.sentEvents.Next(event)
}

func (s *SocketStub) Events() ro.Observable[AnySocketEvent] {
	return s.subject
}

func (s *SocketStub) SentEvents() ro.Observable[AnySocketEvent] {
	return s.sentEvents
}

func (s *SocketStub) Close() {
	s.subject.Complete()
}

func (s *SocketStub) CloseWithError(err error) {
	s.subject.Error(err)
}

func (s *SocketStub) Post(name string, data any) {
	s.subject.Next(AnySocketEvent{name, data, s})
}
