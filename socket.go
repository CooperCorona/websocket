package websocket

import (
	"errors"
	"time"

	"github.com/samber/ro"
)

var (
	ErrSocketClosed = errors.New("cannot send to closed socket")
)

type Socket interface {
	Send(AnySocketEvent)

	Events() ro.Observable[AnySocketEvent]

	// Closes the client. May block until the client is closed.
	Close()
}

// SocketStub is a Socket you can manually send events to.
type SocketStub struct {
	subject           ro.Subject[AnySocketEvent]
	sentEvents        ro.Subject[AnySocketEvent]
	PanicOnClosedSend bool
}

type ConfigurationOptions struct {
	BufferSize int
}

func NewStub() *SocketStub {
	return NewStubWithOptions(ConfigurationOptions{})
}

func NewStubWithOptions(options ConfigurationOptions) *SocketStub {
	return &SocketStub{ro.NewSubject[AnySocketEvent](), ro.NewSubject[AnySocketEvent](), false}
}

func (s *SocketStub) Send(event AnySocketEvent) {
	if s.sentEvents.IsClosed() && s.PanicOnClosedSend {
		panic(ErrSocketClosed)
	}
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
	s.sentEvents.Complete()
}

func (s *SocketStub) CloseWithError(err error) {
	s.subject.Error(err)
}

func (s *SocketStub) Post(name string, data any) {
	s.subject.Next(AnySocketEvent{name, data, s})
}

// Manually emits a message and sleeps for 0.1 seconds.
// Useful for tests where it's crucial for the event loop to
// allow observers to receive the message before the test proceeds.
func (s *SocketStub) PostAndSleep(name string, data any) {
	s.subject.Next(AnySocketEvent{name, data, s})
	time.Sleep(time.Second / 10.0)
}
