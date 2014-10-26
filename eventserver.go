package main

import (
	"github.com/msparks/iq/public"
	"log"
)

type EventType string

type EventServer struct {
	Event chan *public.Event
}

func NewEventServer() *EventServer {
	s := &EventServer{Event: make(chan *public.Event)}
	go readEvents(s)
	return s
}

func readEvents(s *EventServer) {
	log.Print("readEvents started.")
	for {
		ev := <-s.Event
		log.Printf("New event: %+v", ev.String())
	}
}
