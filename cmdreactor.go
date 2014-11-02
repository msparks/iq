package main

import "github.com/msparks/iq/public"

func CommandReactor(evs *EventServer, ns *NamedSession) {
	notifiee := evs.NewNotifiee()
	defer evs.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case *public.Command:
			ircMsg := v.GetIrcMessage()
			if ircMsg != nil && ircMsg.GetHandle() == ns.Handle {
				msg := ircMsg.GetMessage()
				if msg != nil {
					ns.Conn.OutgoingMessageIs(msg)
				}
			}
		}
	}
}
