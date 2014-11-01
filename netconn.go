package main

import "code.google.com/p/goprotobuf/proto"
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
	Notifier

	Network *Network

	state NetworkConnectionState
	conn *irc.Conn
	handle ConnectionHandle
	quit chan bool
	wg sync.WaitGroup
}

type NetworkConnectionStateChange struct {}
type NetworkConnectionEvent struct {
	Event *public.Event
}

func NewNetworkConnection(n *Network) *NetworkConnection {
	nc := &NetworkConnection{Network: n}
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

func (nc *NetworkConnection) State() NetworkConnectionState {
	return nc.state
}

func (nc *NetworkConnection) setState(s NetworkConnectionState) {
	if s != nc.state {
		nc.state = s
		nc.notify(NetworkConnectionStateChange{})
	}
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
	nc.setState(DISCONNECTED)

	for {
		var err error
		for {
			nc.setState(CONNECTING)
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

		nc.setState(CONNECTED)
		log.Printf("[%s] Connected to %s. Handle: %s",
			nc.Network.Name, nc.Network.Config.Server, nc.handle)
		nc.runLoop()
		nc.setState(DISCONNECTED)
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

		p, err := MessageAsProto(message)
		if err != nil {
			continue
		}
		ev := &public.Event{IrcMessage: &public.IrcMessage{
			Handle: proto.String(string(nc.handle)),
			Message: p}}
		nc.notify(NetworkConnectionEvent{Event: ev})

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
