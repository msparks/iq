package main

import "code.google.com/p/goprotobuf/proto"
import "github.com/msparks/iq/notify"
import "github.com/msparks/iq/public"
import ircproto "github.com/msparks/iq/public/irc"
import "github.com/msparks/iq/ircconnection"
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
	notify.Notifier

	Network *Network

	ic *ircconnection.IRCConnection
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

	nc := &NetworkConnection{
		Network: n,
		ic: ircconnection.NewIRCConnection([]ircconnection.Endpoint{endpoint}),
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
	nc.ic.OutgoingMessageIs(p)
}

func (nc *NetworkConnection) run() {
	log.Print("NetworkConnection: run")
	notifiee := nc.ic.NewNotifiee()
	defer nc.ic.CloseNotifiee(notifiee)

	go nc.ic.StateIs(ircconnection.CONNECTING)

	for {
		v := <-notifiee
		log.Printf("NetworkConnection: notification %T", v)
		switch v := v.(type) {
		case ircconnection.StateChange:
			switch nc.ic.State() {
			case ircconnection.DISCONNECTED:
				nc.setState(DISCONNECTED)
				time.Sleep(5 * time.Second)
				log.Printf("Reconnecting...")
				nc.setState(CONNECTING)
				go nc.ic.StateIs(ircconnection.CONNECTING)

			case ircconnection.CONNECTED:
				nc.setState(CONNECTED)
				log.Printf("Connected")
				nc.handle = ConnectionHandle(strconv.FormatInt(rand.Int63(), 16))
				// TODO(msparks): Write USER.
				nick := &ircproto.Message{
					Type: ircproto.Message_NICK.Enum(),
					Nick: &ircproto.Nick{
						NewNick: proto.String(nc.Network.Config.Nick),
					},
				}
				user := &ircproto.Message{
					Type: ircproto.Message_USER.Enum(),
					User: &ircproto.User{
						User: proto.String("IQ"),
						Realname: proto.String("IQ"),
					},
				}
				nc.ic.OutgoingMessageIs(nick)
				nc.ic.OutgoingMessageIs(user)
			}

		case ircconnection.IncomingMessage:
			ev := &public.Event{
				IrcMessage: &public.IrcMessage{
					Handle: proto.String(string(nc.handle)),
					Message: v.Message,
				},
			}

			log.Printf("Incoming event: %+v", ev)

			// TODO(msparks): If welcome message received, join channels.

			nc.Notify(NetworkConnectionEvent{Event: ev})
		}
	}
}
