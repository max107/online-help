package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
)

var (
	homeTempl = template.Must(template.ParseFiles(filepath.Join("templates", "home.html")))
)

func homeHandler(c http.ResponseWriter, req *http.Request) {
	homeTempl.Execute(c, req.Host)
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *connection) reader() {
	for {
		message := new(Message)
		err := c.ws.ReadJSON(&message)
		if err != nil {
			break
		}
		log.Printf("%s", message)
		collections[message.From].broadcast <- message
	}

	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &connection{
		send: make(chan []byte, 256),
		ws:   conn,
	}

	message := new(Message)
	errMsg := conn.ReadJSON(&message)
	if errMsg != nil {
		panic(err)
	}

	if _, h := collections[message.From]; h {
		collections[message.From].register <- c

		defer func() { collections[message.From].unregister <- c }()
		go c.writer()
		c.reader()
	} else {
		h := new(Hub)
		h.broadcast = make(chan *Message)
		h.register = make(chan *connection)
		h.unregister = make(chan *connection)
		h.connections = make(map[*connection]bool)
		go h.run()

		collections[message.From] = h

		h.register <- c

		defer func() { h.unregister <- c }()
		go c.writer()
		c.reader()
	}
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/ws", wsHandler)

	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("assets/"))))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
