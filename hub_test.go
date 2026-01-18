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

type User struct {
	id string
}

func (u User) ID() string {
	return u.id
}

type CrepeEvent struct {
	Name string
}

type OrderFulfilledEvent struct {
	Names []string
}

type Order struct {
	Name string
	User string
}

func TestCrepes(t *testing.T) {
	hub := NewHub[User]()
	stub1 := NewStub(ConfigurationOptions{})
	stub2 := NewStub(ConfigurationOptions{})
	hub.Register(stub1)
	hub.Register(stub2)
	hub.SetUserInfo(stub1, User{"A"})
	hub.SetUserInfo(stub2, User{"B"})
	orders := make(map[string][]Order)
	s := Listen[CrepeEvent]("CrepeEvent")(hub.Events()).Subscribe(ro.OnNext(func(e SocketEvent[CrepeEvent]) {
		user, err := hub.GetUserInfo(e.Socket)
		if err != nil {
			fmt.Printf("ERR: no user info: %v\n", err)
		}
		o := Order{e.Data.Name, user.ID()}
		fmt.Printf("Pushed: %+v\n", o)
		if ords, ok := orders[user.ID()]; ok {
			orders[user.ID()] = append(ords, o)
		} else {
			orders[user.ID()] = []Order{o}
		}
	}))
	// Listen doesn't work because it's not a SocketEvent. Maybe we need to make it a SocketEvent and just discard
	// the socket, allowing it to be nil.
	Listen[OrderFulfilledEvent]("OrderFulfilledEvent")(stub1.SentEvents()).Subscribe(ro.OnNext(func(e SocketEvent[OrderFulfilledEvent]) {
		fmt.Printf("stub1 order fulfilled: %+v\n", e.Data)
	}))
	Listen[OrderFulfilledEvent]("OrderFulfilledEvent")(stub2.SentEvents()).Subscribe(ro.OnNext(func(e SocketEvent[OrderFulfilledEvent]) {
		fmt.Printf("stub2 order fulfilled: %+v\n", e.Data)
	}))
	defer s.Unsubscribe()
	pushTicker := time.NewTicker(time.Second)
	popTicker := time.NewTicker(time.Second * 2)
	count := 10
	crepe := 0
outerLoop:
	for {
		select {
		case <-pushTicker.C:
			crepe += 1
			var stub *SocketStub
			if crepe%2 == 0 {
				stub = stub1
			} else {
				stub = stub2
			}
			stub.Post("CrepeEvent", CrepeEvent{fmt.Sprintf("Crepe #%d", crepe)})
		case <-popTicker.C:
			var user string
			if count%2 == 0 {
				user = "A"
			} else {
				user = "B"
			}
			fmt.Printf("Consuming order for %s. Count: %d\n", user, len(orders[user]))
			names := make([]string, len(orders[user]))
			for i, u := range orders[user] {
				names[i] = u.Name
			}
			orders[user] = []Order{}
			// Want to be able to publish to groups, say, based on role.
			hub.Publish(WithID[User](user), "OrderFulfilledEvent", OrderFulfilledEvent{Names: names})
			// Can do both at once by specifying an individual ID in the filter callback.
			// Hub can handle the callback so the sockets aren't responsible.
			count -= 1
			if count == 0 {
				break outerLoop
			}
		}
	}
}
