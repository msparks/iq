package main

import "log"

func ConnReactor(nc *NetworkConnection, evs *EventServer) {
	notifiee := nc.NewNotifiee()
	defer nc.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case NetworkConnectionStateChange:
			log.Printf("NetworkConnection state changed %s", nc.Network.Name)

		case NetworkConnectionEvent:
			log.Printf("Received event from NetworkConnection: %+v", v)
			evs.Event <-v.Event
		}
	}
}
