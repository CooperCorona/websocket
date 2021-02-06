package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/CooperCorona/websocket"
)

type CallbackClient struct {
	send     chan websocket.Event
	callback func(*CallbackClient, websocket.Event)
}

func (c *CallbackClient) Send() chan<- websocket.Event {
	return c.send
}

func (c *CallbackClient) Close() {
	close(c.send)
}

func (c *CallbackClient) Listen() {
	for {
		select {
		case event := <-c.send:
			c.callback(c, event)
		}
	}
}

func NewClient(callback func(*CallbackClient, websocket.Event)) *CallbackClient {
	return &CallbackClient{
		send:     make(chan websocket.Event, 256),
		callback: callback,
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		tmpl.Execute(w, nil)
	})
	http.HandleFunc("/socket.js", func(w http.ResponseWriter, req *http.Request) {
		websocket.ServeSocketJs(w, req)
		// http.ServeFile(w, req, "socket.js")
	})

	//
	// CONNECT
	//
	websocketConnectHubClosedChannel := make(chan bool, 1)
	websocketConnectHubClosed := false
	http.HandleFunc("/websocket_connect", func(w http.ResponseWriter, req *http.Request) {
		hub := websocket.NewHub()
		hub.CloseOnNoClients = true
		go func() {
			hub.Run()
			websocketConnectHubClosedChannel <- true
		}()
		websocket.ServeWebsocket(hub, w, req, nil)
	})
	http.HandleFunc("/websocket_connect_closed", func(w http.ResponseWriter, req *http.Request) {
		select {
		case result := <-websocketConnectHubClosedChannel:
			websocketConnectHubClosed = result
		default:
		}
		if websocketConnectHubClosed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	//
	// CLOSE
	//
	websocketCloseHubClosedChannel := make(chan bool, 1)
	websocketCloseHubClosed := false
	http.HandleFunc("/websocket_close", func(w http.ResponseWriter, req *http.Request) {
		hub := websocket.NewHub()
		hub.CloseOnNoClients = true
		go func() {
			hub.Run()
			websocketCloseHubClosedChannel <- true
		}()
		websocket.ServeWebsocket(hub, w, req, nil)
	})
	http.HandleFunc("/websocket_close_closed", func(w http.ResponseWriter, req *http.Request) {
		select {
		case result := <-websocketCloseHubClosedChannel:
			websocketCloseHubClosed = result
		default:
		}
		if websocketCloseHubClosed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	//
	// SEND
	//
	websocketSendHubClosedChannel := make(chan bool, 1)
	websocketSendHubClosed := false
	http.HandleFunc("/websocket_send", func(w http.ResponseWriter, req *http.Request) {
		hub := websocket.NewHub()
		go hub.Run()
		websocket.ServeWebsocket(hub, w, req, func(h *websocket.Hub) {
			go hub.Close()
		})
		client := NewClient(func(c *CallbackClient, event websocket.Event) {
			var response struct {
				Text string `json:"text"`
			}
			response.Text = "responded"
			b, _ := json.Marshal(response)
			hub.Broadcast(c, "response", b)
		})
		go client.Listen()
		hub.Register(client, websocket.ClientRegistrationOptions{OnClose: func(h *websocket.Hub) {
			websocketSendHubClosedChannel <- true
		}})
	})
	http.HandleFunc("/websocket_send_closed", func(w http.ResponseWriter, req *http.Request) {
		select {
		case result := <-websocketSendHubClosedChannel:
			websocketSendHubClosed = result
		default:
		}
		if websocketSendHubClosed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	//
	// TIMEOUT
	//
	websocketTimeoutHubClosedChannel := make(chan bool, 1)
	websocketTimeoutHubClosed := false
	http.HandleFunc("/websocket_timeout", func(w http.ResponseWriter, req *http.Request) {
		hub := websocket.NewHub()
		hub.CloseTimeout = time.Second * 5
		go func() {
			hub.Run()
			websocketTimeoutHubClosedChannel <- true
		}()
		websocket.ServeWebsocket(hub, w, req, nil)
	})
	http.HandleFunc("/websocket_timeout_closed", func(w http.ResponseWriter, req *http.Request) {
		select {
		case result := <-websocketTimeoutHubClosedChannel:
			websocketTimeoutHubClosed = result
		default:
		}
		if websocketTimeoutHubClosed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	//
	// TIMEOUT CHANGE
	//
	websocketTimeoutChangeHubClosedChannel := make(chan bool, 1)
	websocketTimeoutChangeHubClosed := false
	http.HandleFunc("/websocket_timeout_change", func(w http.ResponseWriter, req *http.Request) {
		hub := websocket.NewHub()
		hub.CloseTimeout = time.Second * 5
		go func() {
			hub.Run()
			websocketTimeoutChangeHubClosedChannel <- true
		}()
		updateTimeoutTicker := time.NewTicker(time.Second * 2)
		go func() {
			select {
			case _ = <-updateTimeoutTicker.C:
				hub.CloseTimeout = time.Second * 2
				updateTimeoutTicker.Stop()
			}
		}()
		websocket.ServeWebsocket(hub, w, req, nil)
	})
	http.HandleFunc("/websocket_timeout_change_closed", func(w http.ResponseWriter, req *http.Request) {
		select {
		case result := <-websocketTimeoutChangeHubClosedChannel:
			websocketTimeoutChangeHubClosed = result
		default:
		}
		if websocketTimeoutChangeHubClosed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	http.ListenAndServe(":4000", nil)
}
