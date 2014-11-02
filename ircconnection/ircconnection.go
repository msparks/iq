package ircconnection

import (
	"errors"
	"github.com/sorcix/irc"
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
	Endpoints []Endpoint
	Err error

	state State
	wg sync.WaitGroup
	mu sync.Mutex
}

func NewIRCConnection(endpoints []Endpoint) *IRCConnection {
	ic := &IRCConnection{
		Endpoints: endpoints,
		state: DISCONNECTED,
	}
	return ic
}

func (ic *IRCConnection) State() State {
	return ic.state
}

func (ic *IRCConnection) StateIs(s State) (err error) {
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
		ic.wg.Add(1)
		go ic.wrappedRun()

	case CONNECTED:
		return errors.New("Invalid transition")
	}

	return nil
}

func (ic *IRCConnection) wrappedRun() {
	err := ic.run()
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.state = DISCONNECTED
	ic.Err = err

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
	ic.mu.Unlock()

	log.Print("IRCConnection connected.")

	for {
		message, err := conn.Decode()
		if err != nil {
			return err
		}
		log.Printf("IRCConnection message: %+v", message)
	}
}
