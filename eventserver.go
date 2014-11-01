package main

import (
	"github.com/msparks/iq/public"
	"log"
)

type EventServer struct {
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

// TODO(msparks): Locking.
func (s *EventServer) NewListener() chan interface{} {
	c := make(chan interface{})
	s.listeners = append(s.listeners, c)
	return c
}

func (s *EventServer) CloseListener(listener chan interface{}) {
	var r []chan interface{}
	for _, c := range s.listeners {
		if c != listener {
			r = append(r, c)
		} else {
			close(c)
		}
	}
	s.listeners = r
}

func (s *EventServer) readEvents() {
	log.Print("readEvents started.")

	for {
		ev := <-s.Event

		for _, listener := range s.listeners {
			listener <- ev
		}
	}
}

func printEvents(s *EventServer) {
	listener := s.NewListener()
	defer s.CloseListener(listener)

	for {
		v := <-listener
		switch v := v.(type) {
		case *public.Event:
			log.Printf("New event: %+v", v.String())
		default:
			log.Printf("Unhandled type in printEvents: %T", v)
		}
	}
}
