package main

import "code.google.com/p/goprotobuf/proto"
import "github.com/msparks/iq/public"
import ircproto "github.com/msparks/iq/public/irc"
import "log"

type ProtocolController struct {
	nc *NetworkConnection
}

func NewProtocolController(nc *NetworkConnection) *ProtocolController {
	pc := &ProtocolController{nc: nc}
	go pc.run()
	return pc
}

func (pc *ProtocolController) run() {
	notifiee := pc.nc.NewNotifiee()
	defer pc.nc.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		switch v := v.(type) {
		case NetworkConnectionStateChange:
			log.Printf("ProtocolController: netconn state change.")
		case NetworkConnectionEvent:
			pc.eventIs(v.Event)
		}
	}
}

func (pc *ProtocolController) eventIs(ev *public.Event) {
	msg := ev.GetIrcMessage(); if msg != nil {
		if msg.GetMessage().GetType() == ircproto.Message_PING {
			pc.pong(msg)
		}
	}
}

func (pc *ProtocolController) pong(msg *public.IrcMessage) {
	target := msg.GetMessage().GetPing().GetTarget()
	reply := &ircproto.Message{
		Type: ircproto.Message_PONG.Enum(),
		Pong: &ircproto.Pong{
			Target: proto.String(target),
		},
	}

	pc.nc.Write(reply)
}
