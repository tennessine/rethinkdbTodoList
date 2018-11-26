package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type connection struct {
	ws   *websocket.Conn
	send chan interface{}
}

func (c *connection) reader() {
	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for change := range c.send {
		err := c.ws.WriteJSON(change)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(h hub) http.HandlerFunc {
	log.Println("Starting websocket server")
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		c := &connection{
			send: make(chan interface{}, 256),
			ws:   ws,
		}
		h.register <- c
		defer func() {
			h.unregister <- c
		}()
		go c.writer()
		c.reader()
	}
}
