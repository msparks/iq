package main

import "github.com/msparks/iq/public"
import "log"

func PingReactor(evs *EventServer) {
	notifiee := evs.NewNotifiee()
	defer evs.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case *public.Event:
			log.Printf("PingReactor received event: %+v", v)
		}
	}
}
