package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
)

var (
	homeTempl = template.Must(template.ParseFiles(filepath.Join("templates", "home.html")))
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	homeTempl.Execute(w, r.Host)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	domain := params["domain"]
	log.Printf("Domain: %s", domain)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &Connection{send: make(chan []byte), ws: conn}

	cookie, _ := r.Cookie("online_from")
	from := cookie.Value

	log.Printf("%s", domain+from)

	if h, ok := collections[domain+from]; ok {
		h.register <- c
		defer func() { h.unregister <- c }()
	} else {
		h := new(Hub)
		h.broadcast = make(chan Message)
		h.register = make(chan *Connection)
		h.unregister = make(chan *Connection)
		h.connections = make(map[*Connection]bool)
		go h.run(domain, from)

		collections[domain+from] = h

		h.register <- c
		defer func() { h.unregister <- c }()
	}

	go c.WsWriter(domain)
	c.WsReader(domain, from)
}
