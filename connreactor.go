package main

import "code.google.com/p/goprotobuf/proto"
import "github.com/msparks/iq/ircconnection"
import "github.com/msparks/iq/public"
import "log"

func ConnReactor(ns *NamedSession, evs *EventServer) {
	notifiee := ns.Conn.NewNotifiee()
	defer ns.Conn.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case ircconnection.IncomingMessageNotification:
			ev := &public.Event{
				IrcMessage: &public.IrcMessage{
					Handle: proto.String(ns.Handle),
					Message: v.Message,
				},
			}

			log.Printf("ConnReactor emitting event: %+v", ev)
			evs.Event <-ev
		}
	}
}
