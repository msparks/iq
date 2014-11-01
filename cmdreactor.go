package main

import "github.com/msparks/iq/public"

func CommandReactor(evs *EventServer, nc *NetworkConnection) {
	notifiee := evs.NewNotifiee()
	defer evs.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case *public.Command:
			ircMsg := v.GetIrcMessage(); if ircMsg != nil {
				if ircMsg.GetHandle() != string(nc.Handle()) {
					return
				}
				msg := ircMsg.GetMessage(); if msg != nil {
					nc.Write(msg)
				}
			}
		}
	}
}
