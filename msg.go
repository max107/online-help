package main

import (
	"encoding/json"
)

type Message struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

func (msg *Message) ToString() string {
	return msg.From + ": " + msg.Message
}

func (msg *Message) ToByte() []byte {
	return []byte(msg.ToString())
}

func (msg *Message) ToJson() []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return data
}
