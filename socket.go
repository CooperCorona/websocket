package websocket

import (
	"encoding/json"

	"github.com/samber/ro"
)

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
