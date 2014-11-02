package ircconnection

import (
	"errors"
	"github.com/msparks/iq/notify"
	"github.com/sorcix/irc"
	ircproto "github.com/msparks/iq/public/irc"
	"log"
	"sync"
)

type State string

const (
	DISCONNECTED State = "DISCONNECTED"
	CONNECTING   State = "CONNECTING"
	CONNECTED    State = "CONNECTED"
)

type Endpoint struct {
	Address string
}

type IRCConnection struct {
	notify.Notifier

	Endpoints []Endpoint
	Err error

	state State
	wg sync.WaitGroup
	mu sync.Mutex

	out chan *ircproto.Message
}

// Notification types.
type StateChange struct {}
type IncomingMessage struct {
	Message *ircproto.Message
}

func NewIRCConnection(endpoints []Endpoint) *IRCConnection {
	ic := &IRCConnection{
		Endpoints: endpoints,
		state: DISCONNECTED,
	}
	return ic
}

func (ic *IRCConnection) State() State {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	return ic.state
}

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
		ic.Notify(StateChange{})
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
	ic.out <-p
	return nil
}

func (ic *IRCConnection) wrappedRun() {
	err := ic.run()
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.state = DISCONNECTED
	ic.Err = err
	close(ic.out)
	ic.Notify(StateChange{})

	log.Print("IRCConnection error: %s", err)
}

func (ic *IRCConnection) run() error {
	log.Print("IRCConnection connecting...")

	endpoint := ic.Endpoints[0]
	conn, err := irc.Dial(endpoint.Address)
	if err != nil {
		return err
	}
	defer conn.Close()

	ic.mu.Lock()
	ic.state = CONNECTED
	ic.Notify(StateChange{})
	ic.mu.Unlock()

	log.Print("IRCConnection connected.")

	// Closed after the state is changed to DISCONNECTED so OutgoingMessageIs
	// can know if the channel is open.
	ic.out = make(chan *ircproto.Message)
	go func() {
		for {
			p, ok := <-ic.out; if !ok {
				return
			}

			msg, err := ProtoAsMessage(p); if err != nil {
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

		p, err := MessageAsProto(message)
		if err != nil {
			log.Printf("IRCConnection ignoring message: %+v", message)
			continue
		}

		ic.Notify(IncomingMessage{Message: p})
	}
}
