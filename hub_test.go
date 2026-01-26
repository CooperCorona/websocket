package websocket

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/samber/ro"
	rotesting "github.com/samber/ro/testing"
)

func HubPrintObserver[T any](hub *Hub[T]) ro.Subscription {
	return hub.Events().Subscribe(ro.NewObserver(
		func(value AnySocketEvent) {
			fmt.Printf("hub onNext: %+v\n", value)
		},
		func(err error) {
			fmt.Printf("hub onError: %v\n", err)
		},
		func() {
			fmt.Printf("hub onComplete\n")
		},
	))
}

type TestEvent struct {
	X int
}

func TestHub(t *testing.T) {
	hub := NewAnyHub()
	stub := NewStub()
	s4 := hub.Events().Subscribe(ro.OnComplete[AnySocketEvent](func() {
		fmt.Printf("Hub completed\n")
	}))
	s5 := hub.Events().Subscribe(ro.OnError[AnySocketEvent](func(err error) {
		fmt.Printf("Hub errored: %v\n", err)
	}))
	stub2 := NewStub()
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

func listenAny[T any](eventName string, socket Socket) func(ro.Observable[AnyEvent]) ro.Observable[SocketEvent[T]] {
	return func(o ro.Observable[AnyEvent]) ro.Observable[SocketEvent[T]] {
		return ro.Pipe2(o,
			ro.Map(func(a AnyEvent) AnySocketEvent {
				return AnySocketEvent{Name: a.Name, Data: a.Data, Socket: socket}
			}),
			Listen[T](eventName),
		)
	}
}

func TestCrepes(t *testing.T) {
	fmt.Printf("===== START =====\n")
	hub := NewHub[User]()
	stub1 := NewStub()
	stub2 := NewStub()
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
	s2 := listenAny[OrderFulfilledEvent]("OrderFulfilledEvent", stub1)(stub1.SentEvents()).Subscribe(ro.OnNext(func(e SocketEvent[OrderFulfilledEvent]) {
		fmt.Printf("stub1 order fulfilled: %+v\n", e.Data)
	}))
	s3 := listenAny[OrderFulfilledEvent]("OrderFulfilledEvent", stub1)(stub2.SentEvents()).Subscribe(ro.OnNext(func(e SocketEvent[OrderFulfilledEvent]) {
		fmt.Printf("stub2 order fulfilled: %+v\n", e.Data)
	}))
	defer s.Unsubscribe()
	defer s2.Unsubscribe()
	defer s3.Unsubscribe()
	pushTicker := time.NewTicker(time.Second)
	popTicker := time.NewTicker(time.Second * 2)
	count := 2
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
				stub1.Close()
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

func TestStress(t *testing.T) {
	hub := NewHub[User]()
	stub1 := NewStub()
	stub2 := NewStub()
	stub3 := NewStub()
	testStream := ro.Pipe3(
		ro.Merge(
			hub.Events(),
			ToAnySocketEvent(stub1)(stub1.SentEvents()),
			ToAnySocketEvent(stub2)(stub2.SentEvents()),
			ToAnySocketEvent(stub3)(stub3.SentEvents())),
		ro.Filter(func(e AnySocketEvent) bool {
			_, ok := e.Data.(TestEvent)
			return ok
		}),
		ro.Map(func(e AnySocketEvent) int {
			return e.Data.(TestEvent).X
		}),
		ro.ShareReplay[int](1000),
	)
	defer testStream.Subscribe(ro.PrintObserver[int]()).Unsubscribe()

	stub1.PanicOnClosedSend = true
	hub.Register(stub1)
	hub.Register(stub2)
	hub.RegisterWithOptions(stub3, NewRegistrationOptionsForEvents[User]("HubEvent"))
	// allow hub to register
	time.Sleep(time.Second / 10.0)
	// printSub := HubPrintObserver(hub)
	// defer printSub.Unsubscribe()
	stub1.SentEvents().Subscribe(ro.OnNext(func(e AnyEvent) {
		fmt.Printf("Stub1: %+v\n", e)
		hub.Close()
	}))
	stub2.SentEvents().Subscribe(ro.OnNext(func(e AnyEvent) {
		fmt.Printf("Stub2: %+v\n", e)
		hub.Close()
	}))
	stub3.SentEvents().Subscribe(ro.OnNext(func(e AnyEvent) {
		fmt.Printf("Stub3: %+v\n", e)
		hub.Close()
	}))
	stub1.Post("TestEvent", TestEvent{10})
	stub1.Post("TestEvent", TestEvent{20})
	hub.Broadcast("HubEvent", TestEvent{30})
	// Broadcasting can occur on different threads, so we have to wait to ensure the events appear in the same order.
	time.Sleep(time.Second / 10.0)
	stub1.Post("TestEvent", TestEvent{40})
	stub1.Post("TestEvent", TestEvent{50})
	hub.Close()
	time.Sleep(time.Second / 10.0)

	rotesting.Assert[int](t).Source(testStream).ExpectNextSeq(10, 20, 30, 30, 30).Verify()
}
