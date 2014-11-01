package main

import "code.google.com/p/goprotobuf/proto"
import "github.com/msparks/iq/public"
import ircproto "github.com/msparks/iq/public/irc"
import "log"

func PingReactor(evs *EventServer) {
	notifiee := evs.NewNotifiee()
	defer evs.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case *public.Event:
			msg := v.GetIrcMessage(); if msg != nil {
				if msg.GetMessage().GetType() == ircproto.Message_PING {
					log.Printf("PingReactor received PING: %+v", msg)
					pong(msg, evs)
				}
			}
		}
	}
}

func pong(msg *public.IrcMessage, evs *EventServer) {
	target := msg.GetMessage().GetPing().GetTarget()
	reply := &ircproto.Message{
		Type: ircproto.Message_PONG.Enum(),
		Pong: &ircproto.Pong{
			Target: proto.String(target),
		},
	}

	cmd := &public.Command{
		IrcMessage: &public.IrcMessage{
			Handle: proto.String(msg.GetHandle()),
			Message: reply,
		},
	}

	log.Printf("Pong: %+v", cmd)
}
