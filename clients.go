package main

import "log"
import "net/http"
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
		return
	}

	log.Print("Websocket client connected.")

	if err = conn.WriteMessage(websocket.TextMessage, []byte("foo")); err != nil {
		log.Print(err)
	}

	conn.Close()
}
