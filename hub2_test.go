package websocket

import (
	"fmt"
	"testing"

	"github.com/samber/ro"
)

type TestEvent struct {
	X int
}

func TestCooper(t *testing.T) {
	socket := NewStub(ConfigurationOptions{BufferSize: DefaultBufferSize})
	// I need someway to perform an action, idiomatically. Maybe I write a custom sink.
	// I could add a callback to ListenFor, but that's not idiomatic. Do performs
	// side effects which may be all I need. I don't care if I need to drop the events
	// afterwards.
	//
	// WHOOPS, we don't need a DoOnNext. "Subscribe" is the callback.
	// which brings us back to square 1. are we adding abstractions or not? this is pure ro,
	// but we kind of want websocket abstractions because they're nice.
	subscription := ro.Pipe1(socket.Events(), Listen[TestEvent]("XEvent")).Subscribe(ro.OnNext(func(e TestEvent) {
		fmt.Printf("TestEvent: %+v\n", e)
	}))
	// subscription := Listen(socket, "XEvent", func(e TestEvent) {
	// 	fmt.Printf("TestEvent: %+v\n", e)
	// })
	defer subscription.Unsubscribe()
	socket.Post("XEvent", TestEvent{10})
	socket.Post("XEvent", 15)
	socket.Post("YEvent", TestEvent{20})
	socket.Post("XEvent", TestEvent{30})
	socket.Close()
	subscription.Wait()
}

func TestChanne(t *testing.T) {
	emit := make(chan int, 5)
	o := ro.FromChannel(emit)
	s1 := o.Subscribe(ro.OnNext(func(x int) {
		fmt.Printf("X: %d\n", x)
	}))
	defer s1.Unsubscribe()
	emit <- 1
	s2 := o.Subscribe(ro.OnNext(func(y int) {
		fmt.Printf("Y: %d\n", y)
	}))
	defer s2.Unsubscribe()
	emit <- 2
	emit <- 3
	close(emit)
	s1.Wait()
	s2.Wait()
}
