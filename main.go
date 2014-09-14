package main

import (
	"./xmpp"
	"encoding/json"
	// "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	// "github.com/keep94/weblogs"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	xmppConnections = make(map[string]*xmpp.Client)
)

func GetJabberAlias(username string) string {
	return "Менеджер"
}

func GetJabberClient(domain string, from string) *xmpp.Client {
	if talk, ok := xmppConnections[domain+from]; ok {
		return talk
	} else {
		return InitJabber(domain, from)
	}
}

func InitJabber(domain string, from string) *xmpp.Client {
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

					msg := newMessage(username[0], v.Text, domain)
					msg.From = GetJabberAlias(username[0])

					collections[domain+from].msg(msg)
				}
			}
		}
	}()

	xmppConnections[domain+from] = talk
	return talk
}

func SendMsg(domain string, from string, to string, msg Message) {
	talk := GetJabberClient(domain, from)
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
			// default:
			// 	delete(h.connections, c)
			// 	close(c.send)
		}
	}
}

func (h *Hub) run(domain string, from string) {
	for {
		select {
		case c := <-h.register:
			log.Printf("register")
			h.connections[c] = true

			for _, msg := range getMessages(domain) {
				h.msg(msg)
			}

		case c := <-h.unregister:
			log.Printf("unregister")
			delete(h.connections, c)
			close(c.send)
		case msg := <-h.broadcast:
			log.Printf("broadcast")
			SendMsg(domain, from, "max@xmpp.107.su", msg)
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
	CheckOrigin: func(r *http.Request) bool {
		log.Printf("CheckOrigin")
		return true
	},
}

type Connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *Connection) WsReader(domain string, from string) {
	for {
		msg := Message{}
		err := c.ws.ReadJSON(&msg)
		if err != nil {
			break
		}
		msg.Domain = domain
		saveMessage(msg)
		collections[domain+from].broadcast <- msg
	}
	c.ws.Close()
}

func (c *Connection) WsWriter(domain string) {
	for message := range c.send {
		log.Printf("%s", message)
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	initDatabase()

	router := mux.NewRouter()
	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/{domain}/ws", wsHandler)

	http.Handle("/", router)

	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("assets/"))))

	srv := &http.Server{
		Addr:           ":8080",
		Handler:        Log(http.DefaultServeMux),
		ReadTimeout:    1000 * time.Second,
		WriteTimeout:   1000 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(srv.ListenAndServe())
}
