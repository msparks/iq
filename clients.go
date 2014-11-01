package main

import "log"
import "net/http"
import "github.com/gorilla/websocket"
import "github.com/msparks/iq/public"
import "sync"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveWebsocket(s *EventServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("Websocket client connected.")

	notifiee := s.NewNotifiee()
	var wg sync.WaitGroup

	// Relay events from the EventServer to the client.
	wg.Add(1)
	go func() {
		for {
			t, ok := <-notifiee
			if !ok {
				log.Print("Notifiee closed. Writer returning.")
				return
			}
			ev, ok := t.(*public.Event)
			if !ok {
				log.Print("Received unknown type, skipping.")
				continue
			}

			if err = conn.WriteMessage(websocket.TextMessage, []byte(ev.String())); err != nil {
				log.Print("WriteMessage error: ", err)
				return
			}
		}
	}()

	// Read from the client.
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Print(err)
			break
		}
		log.Printf("Message received from websocket: %s", string(p))
	}

	// Kill writer.
	s.CloseNotifiee(notifiee)
	wg.Wait()

	log.Print("Closing websocket connection.")
	conn.Close()
}
