package main

import (
	"./xmpp"
	"log"
	"strings"
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
