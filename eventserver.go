package main

import "log"

type EventType string

const (
	MessagePublic  EventType = "message:public"
	MessagePrivate EventType = "message:private"
)

type EventServer struct {
	Events chan *Event
}

type Event struct {
	Type EventType
}

func NewEventServer() *EventServer {
	s := &EventServer{Events: make(chan *Event)}
	go readEvents(s)
	return s
}

func readEvents(s *EventServer) {
	log.Print("readEvents started.")
	for {
		ev := <-s.Events
		log.Printf("New event: %+v", ev)
	}
}
