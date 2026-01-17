package websocket

import (
	"github.com/samber/ro"
)

type Socket interface {
	Send(AnyEvent)

	Events() ro.Observable[AnyEvent]

	// Closes the client. May block until the client is closed.
	Close()
}

// SocketStub is a Socket you can manually send events to.
type SocketStub struct {
	subject ro.Subject[AnyEvent]
}

type ConfigurationOptions struct {
	BufferSize int
}

func NewStub(options ConfigurationOptions) *SocketStub {
	return &SocketStub{ro.NewSubject[AnyEvent]()}
}

func (s *SocketStub) Send(event AnyEvent) {
	s.subject.Next(event)
}

func (s *SocketStub) Events() ro.Observable[AnyEvent] {
	return s.subject
}

func (s *SocketStub) Close() {
	s.subject.Complete()
}

func (s *SocketStub) CloseWithError(err error) {
	s.subject.Error(err)
}

func (s *SocketStub) Post(name string, data any) {
	s.Send(AnyEvent{name, data})
}
