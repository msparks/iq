// This package provides a state machine interface to an IRC connection.
//
// Incoming messages and connection state changes are delivered as notifications
// via the notify package.
package ircconnection

import (
	"errors"
	"github.com/msparks/iq/notify"
	ircproto "github.com/msparks/iq/public/irc"
	"github.com/sorcix/irc"
	"log"
	"net"
	"sync"
	"time"
)

// State of the IRC connection.
type State string

// IRC connection states.
const (
	DISCONNECTED State = "DISCONNECTED"
	CONNECTING   State = "CONNECTING"
	CONNECTED    State = "CONNECTED"
)

// An IRC server.
type Endpoint struct {
	// net.Dial format.
	Address string
}

// IRCConnection is a state machine for a connection to an IRC network. Its
// inputs and outputs are both *ircproto.Message types.
//
// The initial state is DISCONNECTED.
type IRCConnection struct {
	// IRCConnection is a notifier. See the Notification types.
	notify.Notifier

	// Servers to connect to.
	Endpoints []Endpoint
	Err error

	state State
	wg sync.WaitGroup
	mu sync.Mutex

	out chan *ircproto.Message
}

// Delivered to notifiees when the IRC connection state changes.
type StateChangeNotification struct {}

// Delivered to notifiees when an IRC message is received from the connection.
type IncomingMessageNotification struct {
	Message *ircproto.Message
}

func NewIRCConnection(endpoints []Endpoint) *IRCConnection {
	ic := &IRCConnection{
		Endpoints: endpoints,
		state:     DISCONNECTED,
	}
	return ic
}

// Returns the current State of the connection.
func (ic *IRCConnection) State() State {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	return ic.state
}

// Changes the State of the connection.
//
// Allowed transitions:
//
//   DISCONNECTED -> CONNECTING
//   CONNECTED -> DISCONNECTED
func (ic *IRCConnection) StateIs(s State) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if s == ic.state {
		// No-op.
		return nil
	}

	switch s {
	case DISCONNECTED:
		// Shut down.
		return errors.New("Disconnection unimplemented.")

	case CONNECTING:
		if ic.state == CONNECTED {
			return errors.New("Invalid transition")
		}
		// Start connecting.
		ic.Err = nil
		ic.state = s
		ic.Notify(StateChangeNotification{})
		ic.wg.Add(1)
		go ic.wrappedRun()

	case CONNECTED:
		return errors.New("Invalid transition")
	}

	return nil
}

func (ic *IRCConnection) OutgoingMessageIs(p *ircproto.Message) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if ic.state != CONNECTED {
		return errors.New("Not connected")
	}
	ic.out <- p
	return nil
}

func (ic *IRCConnection) wrappedRun() {
	ic.out = make(chan *ircproto.Message)
	defer close(ic.out)

	err := ic.run()
	log.Printf("IRCConnection error: %s", err)

	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.state = DISCONNECTED
	ic.Err = err
	ic.Notify(StateChangeNotification{})
}

func (ic *IRCConnection) run() error {
	log.Print("IRCConnection connecting...")

	endpoint := ic.Endpoints[0]
	dialer := &net.Dialer{
		KeepAlive: 10 * time.Second,
	}
	sock, err := dialer.Dial("tcp", endpoint.Address)
	if err != nil {
		return err
	}
	conn := irc.NewConn(sock)
	defer conn.Close()

	ic.mu.Lock()
	ic.state = CONNECTED
	ic.Notify(StateChangeNotification{})
	ic.mu.Unlock()

	log.Print("IRCConnection connected.")

	go func() {
		for {
			p, ok := <-ic.out
			if !ok {
				return
			}

			msg, err := protoAsMessage(p)
			if err != nil {
				log.Printf("Ignoring outgoing message: %+v", p)
				continue
			}
			log.Printf("Sending message: %+v", p)
			err = conn.Encode(msg)
			if err != nil {
				log.Printf("Error sending message: %s", err)
			}
		}
	}()

	for {
		message, err := conn.Decode()
		if err != nil {
			return err
		}

		p, err := messageAsProto(message)
		if err != nil {
			log.Printf("IRCConnection ignoring message: %+v", message)
			continue
		}

		ic.Notify(IncomingMessageNotification{Message: p})
	}
}
