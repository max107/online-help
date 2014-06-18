package main

import (
	"log"
)

var (
	collections = make(map[string]*Hub)
)

type Hub struct {
	// Registered connections.
	connections map[*connection]bool

	// Inbound messages from the connections.
	broadcast chan *Message

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection
}

func (h *Hub) msg(msg *Message) {
	for c := range h.connections {
		select {
		case c.send <- msg.ToJson():
		default:
			delete(h.connections, c)
			close(c.send)
		}
	}
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			log.Printf("register")
			h.connections[c] = true
		case c := <-h.unregister:
			log.Printf("unregister")
			delete(h.connections, c)
			close(c.send)
		case msg := <-h.broadcast:
			log.Printf("broadcast")
			SendMsg("max@xmpp.107.su", msg)
			h.msg(msg)
		}
	}
}
