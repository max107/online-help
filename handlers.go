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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	homeTempl.Execute(w, r.Host)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	params := mux.Vars(r)
	apikey := params["apikey"]
	log.Printf("%s", apikey)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &Connection{send: make(chan []byte), ws: conn}

	// cookie, _ := r.Cookie("online_from")
	// from := cookie.Value

	if h, ok := collections[apikey]; ok {
		h.register <- c
		defer func() { h.unregister <- c }()
	} else {
		h := new(Hub)
		h.broadcast = make(chan Message)
		h.register = make(chan *Connection)
		h.unregister = make(chan *Connection)
		h.connections = make(map[*Connection]bool)
		go h.run(apikey)

		collections[apikey] = h

		h.register <- c
		defer func() { h.unregister <- c }()
	}

	go c.WsWriter()
	c.WsReader(apikey)
}
