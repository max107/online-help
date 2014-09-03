package main

import (
	"./xmpp"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
)

var (
	xmppConnections = make(map[string]*xmpp.Client)
)

func GetJabberAlias(username string) string {
	return "Менеджер"
}

func GetJabberClient(apikey string, from string) *xmpp.Client {
	if talk, ok := xmppConnections[apikey+from]; ok {
		return talk
	} else {
		return InitJabber(apikey, from)
	}
}

func InitJabber(apikey string, from string) *xmpp.Client {
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

					msg := newMessage(username[0], v.Text)
					msg.From = GetJabberAlias(username[0])

					collections[apikey+from].msg(msg)
				}
			}
		}
	}()

	xmppConnections[apikey+from] = talk
	return talk
}

func SendMsg(apikey string, from string, to string, msg Message) {
	talk := GetJabberClient(apikey, from)
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
	broadcast chan Message
	// Register requests from the connections.
	register chan *Connection
	// Unregister requests from connections.
	unregister chan *Connection
}

func (h *Hub) msg(msg Message) {
	for c := range h.connections {
		select {
		case c.send <- msg.ToJson():
		default:
			delete(h.connections, c)
			close(c.send)
		}
	}
}

func (h *Hub) run(apikey string, from string) {
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
			SendMsg(apikey, from, "max@xmpp.107.su", msg)
			h.msg(msg)
		}
	}
}

func (msg Message) ToJson() []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return data
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

func (c *Connection) WsReader(apikey string, from string) {
	for {
		msg := Message{}
		err := c.ws.ReadJSON(&msg)
		if err != nil {
			break
		}
		saveMessage(msg)
		collections[apikey+from].broadcast <- msg
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

func main() {
	initDatabase()

	router := mux.NewRouter()
	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/{apikey:[a-z]+}/ws", wsHandler).Methods("GET")

	http.Handle("/", router)

	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("assets/"))))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
