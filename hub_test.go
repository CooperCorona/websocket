package websocket

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/samber/ro"
)

type TestEvent struct {
	X int
}

func TestCooper(t *testing.T) {
	socket := NewStub(ConfigurationOptions{BufferSize: DefaultBufferSize})
	subscription := ro.Pipe1(socket.Events(), Listen[TestEvent]("XEvent")).Subscribe(ro.OnNext(func(e SocketEvent[TestEvent]) {
		fmt.Printf("TestEvent: %+v\n", e)
	}))
	defer subscription.Unsubscribe()
	socket.Post("XEvent", TestEvent{10})
	socket.Post("XEvent", 15)
	socket.Post("YEvent", TestEvent{20})
	socket.Post("XEvent", TestEvent{30})
	socket.Close()
	subscription.Wait()
}

func TestHub(t *testing.T) {
	hub := NewAnyHub()
	stub := NewStub(ConfigurationOptions{})
	s4 := hub.Events().Subscribe(ro.OnComplete[AnySocketEvent](func() {
		fmt.Printf("Hub completed\n")
	}))
	s5 := hub.Events().Subscribe(ro.OnError[AnySocketEvent](func(err error) {
		fmt.Printf("Hub errored: %v\n", err)
	}))
	stub2 := NewStub(ConfigurationOptions{})
	hub.Register(stub)
	hub.Register(stub2)
	hub.CloseOnNoClients = true
	s1 := hub.Events().Subscribe(ro.OnNext(func(e AnySocketEvent) {
		fmt.Printf("Event: %+v\n", e)
	}))
	defer s1.Unsubscribe()
	defer s4.Unsubscribe()
	defer s5.Unsubscribe()
	stub.Post("TestEvent", TestEvent{10})
	time.Sleep(time.Second)
	stub.Post("TestEvent", TestEvent{20})
	stub.CloseWithError(errors.New("stub error"))
	stub2.Close()
	time.Sleep(time.Second)
}
