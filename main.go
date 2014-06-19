package main

import (
	"./xmpp"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	xmppConnections = make(map[string]*xmpp.Client)
)

func GetJabberAlias(username string) string {
	aliases := map[string]string{
		"max@xmpp.107.su": "Максим",
	}

	if alias, ok := aliases[username]; ok {
		return alias
	} else {
		return username
	}
}

func GetJabberClient(from string) *xmpp.Client {
	if talk, ok := xmppConnections[from]; ok {
		return talk
	} else {
		return InitJabber(from)
	}
}

func InitJabber(from string) *xmpp.Client {
	options := xmpp.Options{
		Host:     "helpdev.info",
		User:     "",
		Password: "",
		NoTLS:    true,
		Debug:    false,
		Session:  false,
	}

	talk, err := options.NewClient()

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			chat, err := talk.Recv()
			if err != nil {
				log.Fatal(err)
			}
			switch v := chat.(type) {
			case xmpp.Chat:
				if v.Text != "" {
					username := strings.Split(v.Remote, "/")

					msg := new(Message)
					msg.From = GetJabberAlias(username[0])
					msg.Message = v.Text

					collections[from].msg(msg)
				}
			}
		}
	}()

	xmppConnections[from] = talk
	return talk
}

func SendMsg(to string, msg *Message) {
	talk := GetJabberClient(msg.From)
	talk.Send(xmpp.Chat{
		Remote: to,
		Type:   "chat",
		Text:   msg.Message,
	})
}

var (
	collections = make(map[string]*Hub)
)

type Hub struct {
	// Registered connections.
	connections map[*Connection]bool
	// Inbound messages from the connections.
	broadcast chan *Message
	// Register requests from the connections.
	register chan *Connection
	// Unregister requests from connections.
	unregister chan *Connection
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

type Message struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

func (msg *Message) ToJson() []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return data
}

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

type Connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *Connection) WsReader() {
	for {
		msg := new(Message)
		err := c.ws.ReadJSON(&msg)
		if err != nil {
			break
		}
		collections[msg.From].broadcast <- msg
	}
	c.ws.Close()
}

func (c *Connection) WsWriter() {
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

	c := &Connection{send: make(chan []byte, 256), ws: conn}

	cookie, _ := r.Cookie("online_from")
	from := cookie.Value

	if h, ok := collections[from]; ok {
		h.register <- c
		defer func() { h.unregister <- c }()
	} else {
		h := new(Hub)
		h.broadcast = make(chan *Message)
		h.register = make(chan *Connection)
		h.unregister = make(chan *Connection)
		h.connections = make(map[*Connection]bool)
		go h.run()

		collections[from] = h

		h.register <- c
		defer func() { h.unregister <- c }()
	}

	go c.WsWriter()
	c.WsReader()
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/ws", wsHandler)

	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("assets/"))))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
