package ircconnection

import "code.google.com/p/goprotobuf/proto"
import "errors"
import ircproto "github.com/msparks/iq/public/irc"
import "github.com/sorcix/irc"

func protoAsMessage(p *ircproto.Message) (message *irc.Message, err error) {
	message = &irc.Message{}

	switch p.GetType() {
	case ircproto.Message_PONG:
		message.Command = irc.PONG
		message.Params = []string{p.GetPong().GetSource()}
		message.Trailing = p.GetPong().GetTarget()

	case ircproto.Message_NICK:
		message.Command = irc.NICK
		message.Params = []string{p.GetNick().GetNewNick()}
		if len(message.Params) == 0 {
			return nil, errors.New("new_nick must be specified")
		}

	case ircproto.Message_USER:
		user := p.GetUser().GetUser(); if user == "" {
			return nil, errors.New("user must be specified")
		}
		message.Command = irc.USER
		// TODO(msparks): Mode.
		message.Params = []string{user, "0", "*", p.GetUser().GetRealname()}

	default:
		return nil, errors.New("Unknown message type")
	}

	return message, nil
}

func messageAsProto(message *irc.Message) (p *ircproto.Message, err error) {
	p = &ircproto.Message{
		Type: ircproto.Message_UNKNOWN.Enum(),
	}

	switch message.Command {
	case irc.PING:
		var source string
		if len(message.Params) > 0 {
			source = message.Params[0]
		}
		p.Type = ircproto.Message_PING.Enum()
		p.Ping = &ircproto.Ping{
			Source: proto.String(source),
			Target: proto.String(message.Trailing),
		}

	case irc.PRIVMSG:
		var target string
		if len(message.Params) > 0 {
			target = message.Params[0]
		}
		p.Type = ircproto.Message_PRIVMSG.Enum()
		p.Privmsg = &ircproto.Privmsg{
			Source:  prefixProto(message.Prefix),
			Target:  proto.String(target),
			Message: proto.String(message.Trailing),
		}

	case irc.NOTICE:
		var target string
		if len(message.Params) > 0 {
			target = message.Params[0]
		}
		p.Type = ircproto.Message_NOTICE.Enum()
		p.Notice = &ircproto.Notice{
			Source:  prefixProto(message.Prefix),
			Target:  proto.String(target),
			Message: proto.String(message.Trailing),
		}

	case irc.NICK:
		if len(message.Params) == 0 {
			return nil, errors.New("Params must be non-empty")
		}
		p.Type = ircproto.Message_NICK.Enum()
		p.Nick = &ircproto.Nick{
			Source: prefixProto(message.Prefix),
			NewNick: proto.String(message.Params[0]),
		}

	default:
		return nil, errors.New("Unknown command")
	}

	return p, nil
}

func prefixProto(prefix *irc.Prefix) (p *ircproto.Prefix) {
	p = &ircproto.Prefix{
		Name: proto.String(prefix.Name),
		User: proto.String(prefix.User),
		Host: proto.String(prefix.Host),
	}
	return p
}
