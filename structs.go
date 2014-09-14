package main

import (
	"time"
)

type Message struct {
	Id        int64     `json:"id"`
	From      string    `sql:"size:255" json:"from"`
	Message   string    `sql:"size:255" json:"message"`
	Domain    string    `sql:"size:255"`
	CreatedAt time.Time `json:"created_at"`
}
