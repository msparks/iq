package main

import (
	"github.com/msparks/iq/notify"
	"github.com/msparks/iq/public"
	"log"
)

type EventServer struct {
	notify.Notifier

	Event chan *public.Event
	Command chan *public.Command
}

func NewEventServer() *EventServer {
	s := &EventServer{
		Event: make(chan *public.Event),
		Command: make(chan *public.Command),
	}
	go s.readChannels()

	// For debugging.
	go printEvents(s)

	return s
}

func (s *EventServer) readChannels() {
	for {
		select {
		case ev := <-s.Event:
			s.Notify(ev)
		case cmd := <-s.Command:
			s.Notify(cmd)
		}
	}
}

// TODO(msparks): Make this into a reactor elsewhere.
func printEvents(s *EventServer) {
	notifiee := s.NewNotifiee()
	defer s.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case *public.Event:
			log.Printf("New event: %+v", v.String())
		case *public.Command:
			log.Printf("New command: %+v", v.String())
		default:
			log.Printf("Unhandled type in printEvents: %T", v)
		}
	}
}
