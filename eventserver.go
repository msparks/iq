package main

import (
	"github.com/msparks/iq/public"
	"log"
)

type EventServer struct {
	Notifier

	Event chan *public.Event

	listeners []chan interface{}
}

func NewEventServer() *EventServer {
	s := &EventServer{Event: make(chan *public.Event)}
	go s.readEvents()

	// For debugging.
	go printEvents(s)

	return s
}

func (s *EventServer) readEvents() {
	log.Print("readEvents started.")

	for {
		ev := <-s.Event
		s.notify(ev)
	}
}

func printEvents(s *EventServer) {
	notifiee := s.NewNotifiee()
	defer s.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case *public.Event:
			log.Printf("New event: %+v", v.String())
		default:
			log.Printf("Unhandled type in printEvents: %T", v)
		}
	}
}
