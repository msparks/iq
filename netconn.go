package main

import "code.google.com/p/goprotobuf/proto"
import "errors"
import "github.com/msparks/iq/public"
import ircproto "github.com/msparks/iq/public/irc"
import "github.com/sorcix/irc"
import "log"
import "math/rand"
import "strconv"
import "sync"
import "time"

type ConnectionHandle string

type NetworkConnectionState int
const (
	DISCONNECTED NetworkConnectionState = iota
	CONNECTING
	CONNECTED
)

type NetworkConnection struct {
	Network *Network

	evs *EventServer
	state NetworkConnectionState
	conn *irc.Conn
	handle ConnectionHandle
	quit chan bool
	wg sync.WaitGroup
}

func NewNetworkConnection(n *Network, evs *EventServer) *NetworkConnection {
	nc := &NetworkConnection{Network: n, evs: evs}
	nc.quit = make(chan bool)
	nc.wg.Add(1)
	go nc.connectLoop()
	return nc
}

func (nc *NetworkConnection) Stop() {
	nc.quit <-true
}

func (nc *NetworkConnection) Wait() {
	nc.wg.Wait()
}

func (nc *NetworkConnection) write(m *irc.Message) {
	log.Printf("[%s] >> %v", nc.Network.Name, m)
	err := nc.conn.Encode(m)
	if err != nil {
		log.Fatal(err)
	}
}

func (nc *NetworkConnection) connectLoop() {
	log.Printf("Network %s: config=%+v", nc.Network.Name, nc.Network.Config)
	nc.state = DISCONNECTED

	for {
		var err error
		for {
			nc.state = CONNECTING
			nc.handle = ConnectionHandle(strconv.FormatInt(rand.Int63(), 16))
			log.Printf("[%s] Connecting to %s.", nc.Network.Name, nc.Network.Config.Server)

			nc.conn, err = irc.Dial(nc.Network.Config.Server)
			if err != nil {
				log.Printf(
					"[%s] Connection error: %v. Retrying in 5 seconds.",
					nc.Network.Name, err)
				time.Sleep(5 * time.Second)
			} else {
				break
			}
		}

		nc.state = CONNECTED
		log.Printf("[%s] Connected to %s. Handle: %s",
			nc.Network.Name, nc.Network.Config.Server, nc.handle)
		nc.runLoop()
		nc.state = DISCONNECTED
		log.Printf("[%s] Disconnected from %s.", nc.Network.Name, nc.Network.Config.Server)
		time.Sleep(10 * time.Second)
	}
}

func (nc *NetworkConnection) runLoop() {
	var authed bool
	for {
		message, err := nc.conn.Decode()
		if err != nil {
			return
		}

		log.Printf(
			"[%s] %v %v %v %v",
			nc.Network.Name,
			message.Prefix,
			message.Command,
			message.Params,
			message.Trailing)

		if !authed {
			nick := &irc.Message{nil, irc.NICK, []string{nc.Network.Config.Nick}, ""}
			user := &irc.Message{nil, irc.USER, []string{"iq", "*", "*"}, "IQ"}

			nc.write(nick)
			nc.write(user)

			authed = true
		}

		if message.Command == irc.RPL_WELCOME {
			// We're connected. Join configured channels.
			for _, channel := range nc.Network.Channels {
				join := &irc.Message{nil, irc.JOIN, []string{channel.Name}, ""}
				nc.write(join)
			}
		}

		p, err := ircProtoMessage(message)
		if err != nil {
			continue
		}
		ev := &public.Event{IrcMessage: &public.IrcMessage{
			Handle: proto.String(string(nc.handle)),
			Message: p}}
		nc.evs.Event <- ev

		if p.GetType() == ircproto.Message_PING {
			ping := p.GetPing()
			var params []string
			if ping.GetSource() != "" {
				params = append(params, ping.GetSource())
			}
			pong := &irc.Message{nil, irc.PONG, params, ping.GetTarget()}
			nc.write(pong)
		}
	}
}

func prefixProto(prefix *irc.Prefix) (p *ircproto.Prefix) {
	p = &ircproto.Prefix{
		Name: proto.String(prefix.Name),
		User: proto.String(prefix.User),
		Host: proto.String(prefix.Host),
	}
	return p
}

func ircProtoMessage(message *irc.Message) (p *ircproto.Message, err error) {
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

	default:
		return nil, errors.New("Unknown command")
	}

	return p, nil
}
