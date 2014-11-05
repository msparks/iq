package ircconnection

import (
	"bufio"
	"code.google.com/p/goprotobuf/proto"
	ircproto "github.com/msparks/iq/public/irc"
	. "gopkg.in/check.v1"
	"io"
	"net"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type IRCConnectionTest struct{}

var _ = Suite(&IRCConnectionTest{})

// Returns a listening local server for testing.
func localServer(c *C) net.Listener {
	// Local server.
	server, err := net.Listen("tcp", "[::]:0")
	c.Assert(err, IsNil)
	c.Logf("Listening on %s.", server.Addr().String())
	return server
}

func (s *IRCConnectionTest) TestNewIRCConnection(c *C) {
	ep := Endpoint{"some server"}
	ic := NewIRCConnection([]Endpoint{ep})

	c.Check(ic.State(), Equals, DISCONNECTED)
	c.Assert(len(ic.Endpoints), Equals, 1)
	c.Check(ic.Endpoints[0], Equals, ep)
}

func (s *IRCConnectionTest) TestFromRWC(c *C) {
	server := localServer(c)

	// Connect to local server.
	conn, err := net.Dial("tcp", server.Addr().String())
	c.Assert(err, IsNil)

	// Create an IRCConnection from the established connection.
	ep := Endpoint{"some server"}
	ic := FromRWC(conn, []Endpoint{ep})

	c.Check(ic.State(), Equals, CONNECTED)
}

func (s *IRCConnectionTest) TestConnect(c *C) {
	server := localServer(c)

	ep := Endpoint{server.Addr().String()}
	ic := NewIRCConnection([]Endpoint{ep})

	// Start connecting. Our server won't accept the connection until we
	// explicitly call Accept, so there is no race here.
	c.Check(ic.State(), Equals, DISCONNECTED)
	c.Assert(ic.StateIs(CONNECTING), IsNil)
	c.Check(ic.State(), Equals, CONNECTING)

	// Accept the connection. The state changes asynchronously, so we use
	// notifications to wait for it.
	notifiee := ic.NewNotifiee()
	defer ic.CloseNotifiee(notifiee)
	peer, err := server.Accept()
	defer peer.Close()
	c.Assert(err, IsNil)
	<-notifiee
	c.Check(ic.State(), Equals, CONNECTED)
}

func (s *IRCConnectionTest) TestRead(c *C) {
	server := localServer(c)

	ep := Endpoint{server.Addr().String()}
	ic := NewIRCConnection([]Endpoint{ep})
	c.Assert(ic.StateIs(CONNECTING), IsNil)

	notifiee := ic.NewNotifiee()
	defer ic.CloseNotifiee(notifiee)

	// Wait for the client to connect, then write some data to it.
	go func() {
		peer, _ := server.Accept()
		io.WriteString(peer, "PING :foo\r\n")
	}()

	// Wait for the incoming message notifications.
	for {
		v := <-notifiee
		switch v := v.(type) {
		case IncomingMessageNotification:
			c.Assert(v.Message.GetType(), Equals, ircproto.Message_PING)
			return
		}
	}
}

func (s *IRCConnectionTest) TestWrite(c *C) {
	server := localServer(c)

	ep := Endpoint{server.Addr().String()}
	ic := NewIRCConnection([]Endpoint{ep})
	c.Assert(ic.StateIs(CONNECTING), IsNil)

	pong := &ircproto.Message{
		Type: ircproto.Message_PONG.Enum(),
		Pong: &ircproto.Pong{
			Source: proto.String("source"),
			Target: proto.String("target"),
		},
	}

	// Can't send messages yet; we're not connected.
	err := ic.OutgoingMessageIs(pong)
	c.Check(err, ErrorMatches, "Not connected")

	// Accept the connection and wait for the client to notice.
	notifiee := ic.NewNotifiee()
	defer ic.CloseNotifiee(notifiee)
	peer, _ := server.Accept()
	<-notifiee

	// Try sending again.
	err = ic.OutgoingMessageIs(pong)
	c.Check(err, IsNil)

	// Verify the message from the peer.
	reader := bufio.NewReader(peer)
	line, err := reader.ReadString('\n')
	c.Assert(err, IsNil)
	c.Check(line, Equals, "PONG source :target\r\n")
}
