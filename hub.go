package websocket

import (
	"encoding/json"
	"time"
)

// Event is a message sent to a Hub. It represents a json
// object with two fields. Name is a string defining the
// type of event, and Data is an arbitrary object containing
// event data. It is the responsibility of the consumer to
// convert data into its expected format.
type Event struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

// ClientRegistrationOptions configure a Client when registering it with a Hub.
type ClientRegistrationOptions struct {
	ReceiveSelfMessages bool
	OnClose             func(*Hub)
}

// clientData encapsulates a Client and its configuration in a Hub.
type clientData struct {
	// client is the Client associated with this object.
	client Client
	// receiveSelfMessages is true if client should receive messages from
	// the Hub this client itself has sent, false if the Hub should not
	// send this client its messages.
	receiveSelfMessages bool
	// onClose is a function executed when the Hub unregisters the client
	// and closes the client's Send channel.
	onClose func(*Hub)
}

// A message sent from a specific Client.
type clientMessage struct {
	// client is the Client which sent the message.
	client Client
	// message is the message sent by the client.
	event Event
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// clients are the registered clients.
	clients map[Client]clientData

	// braodcast is the inbound messages from the clients.
	broadcast chan clientMessage

	// register receives register requests from the clients.
	register chan clientData

	// unregister receives unregister requests from clients.
	unregister chan Client

	// close closes the Hub
	close chan bool

	// Closes the hub when no clients are remaining. Does not close the hub
	// unless there previously was a client (does not close immediately if
	// no clients have connected yet).
	CloseOnNoClients bool

	// true if a client has been registered (even if it is not anymore), false
	// if 0 clients have been registered.
	clientsHaveExisted bool

	// CloseTimeout is timeout period. If no messages are sent for this
	// amount of time, the Hub closes automatically. Defaults to 10 minutes.
	// Must be positive.
	CloseTimeout time.Duration

	// The time the last message was sent. Defaults to the time the hub
	// began listening for messages.
	lastMessageTimestamp time.Time
}

// NewHub constructs a new hub with empty values.
func NewHub() *Hub {
	return &Hub{
		broadcast:          make(chan clientMessage),
		register:           make(chan clientData),
		unregister:         make(chan Client),
		clients:            make(map[Client]clientData),
		close:              make(chan bool),
		CloseOnNoClients:   false,
		clientsHaveExisted: false,
		CloseTimeout:       time.Minute * 10,
	}
}

// Broadcast sends a message from a client to all registered clients.
// Blocks until the message is broadcasted.
func (h *Hub) Broadcast(client Client, event string, b []byte) {
	h.broadcast <- clientMessage{client, Event{event, b}}
}

// Register registers a client with the given options to receive messages.
// Blocks until the client is registered.
func (h *Hub) Register(client Client, options ClientRegistrationOptions) {
	h.register <- clientData{client, options.ReceiveSelfMessages, options.OnClose}
}

// Unregister removes a client. Blocks until the client is unregistered.
func (h *Hub) Unregister(client Client) {
	h.unregister <- client
}

// Close closes the hub and all registered clients. Blocks until the hub is closed.
func (h *Hub) Close() {
	h.close <- true
}

func (h *Hub) closeClient(client Client, data clientData) {
	delete(h.clients, client)
	client.Close()
	if data.onClose != nil {
		data.onClose(h)
	}
}

func (h *Hub) closeAllClients() {
	for client, clientData := range h.clients {
		h.closeClient(client, clientData)
	}
}

// Run listens for register, unregister, broadcast, and close events.
// Blocks while the hub is running. Run on a separate goroutine
// if you do not wish to block.
func (h *Hub) Run() {
	defer h.closeAllClients()
	timeoutTicker := time.NewTicker(h.CloseTimeout)
	defer timeoutTicker.Stop()
	for {
		select {
		case clientData := <-h.register:
			h.clients[clientData.client] = clientData
			h.clientsHaveExisted = true
		case client := <-h.unregister:
			if clientData, ok := h.clients[client]; ok {
				h.closeClient(client, clientData)
			}
			if len(h.clients) == 0 && h.CloseOnNoClients && h.clientsHaveExisted {
				return
			}
		case clientMessage := <-h.broadcast:
			h.lastMessageTimestamp = time.Now()
			for client, clientData := range h.clients {
				if !clientData.receiveSelfMessages && clientMessage.client == client {
					// This message was sent by the current client, but the current
					// client does not receive its own messages. Skip it.
					continue
				}
				select {
				case client.Send() <- clientMessage.event:
				default:
					h.closeClient(client, clientData)
				}
			}
			if len(h.clients) == 0 && h.CloseOnNoClients && h.clientsHaveExisted {
				return
			}
		case _ = <-h.close:
			return
		case _ = <-timeoutTicker.C:
			if time.Now().Sub(h.lastMessageTimestamp) >= h.CloseTimeout {
				return
			}
			timeoutTicker.Stop()
			timeoutTicker = time.NewTicker(h.CloseTimeout)
		}
	}
}
