package main

import "code.google.com/p/goprotobuf/proto"
import "github.com/msparks/iq/notify"
import "github.com/msparks/iq/public"
import ircproto "github.com/msparks/iq/public/irc"
import "github.com/msparks/iq/ircconnection"
import "github.com/msparks/iq/ircsession"
import "github.com/sorcix/irc"
import "log"
import "math/rand"
import "strconv"
import "sync"

type ConnectionHandle string

type NetworkConnectionState int
const (
	DISCONNECTED NetworkConnectionState = iota
	CONNECTING
	CONNECTED
)

type NetworkConnection struct {
	notify.Notifier

	Network *Network

	session *ircsession.IRCSession

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
	endpoint := ircconnection.Endpoint{Address: n.Config.Server}

	settings := ircsession.IRCSettings{
		Nicknames: []string{n.Config.Nick},
		User: "IQ",
		Realname: "IQ",
	}

	conn := ircconnection.NewIRCConnection([]ircconnection.Endpoint{endpoint})
	session := ircsession.NewIRCSession(settings, conn)

	nc := &NetworkConnection{
		Network: n,
		session: session,
	}

	nc.quit = make(chan bool)
	nc.wg.Add(1)
	go nc.run()
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
		nc.Notify(NetworkConnectionStateChange{})
	}
}

// TODO(msparks): Locking.
func (nc *NetworkConnection) Handle() ConnectionHandle {
	return nc.handle
}

// TODO(msparks): Return error.
func (nc *NetworkConnection) Write(p *ircproto.Message) {
	nc.session.Conn.OutgoingMessageIs(p)
}

func (nc *NetworkConnection) run() {
	log.Print("NetworkConnection: run")

	connNotifiee := nc.session.Conn.NewNotifiee()
	defer nc.session.Conn.CloseNotifiee(connNotifiee)

	nc.handle = ConnectionHandle(strconv.FormatInt(rand.Int63(), 16))

	go nc.session.Conn.StateIs(ircconnection.CONNECTING)

	for {
		v := <-connNotifiee
		log.Printf("NetworkConnection: notification %T", v)
		switch v := v.(type) {
		case ircconnection.IncomingMessageNotification:
			ev := &public.Event{
				IrcMessage: &public.IrcMessage{
					Handle: proto.String(string(nc.handle)),
					Message: v.Message,
				},
			}

			log.Printf("Incoming event: %+v", ev)

			nc.Notify(NetworkConnectionEvent{Event: ev})
		}
	}
}
