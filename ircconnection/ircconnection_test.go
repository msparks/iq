package ircconnection

import (
	. "gopkg.in/check.v1"
	"net"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type IRCConnectionTest struct{}

var _ = Suite(&IRCConnectionTest{})

func (s *IRCConnectionTest) TestNewIRCConnection(c *C) {
	ep := Endpoint{"some server"}
	ic := NewIRCConnection([]Endpoint{ep})

	c.Check(ic.State(), Equals, DISCONNECTED)
	c.Assert(len(ic.Endpoints), Equals, 1)
	c.Check(ic.Endpoints[0], Equals, ep)
}

func (s *IRCConnectionTest) TestFromRWC(c *C) {
	// Local server.
	server, err := net.Listen("tcp", "[::]:0")
	c.Assert(err, IsNil)
	defer server.Close()
	c.Logf("Listening on %s.", server.Addr().String())

	// Connect to local server.
	conn, err := net.Dial("tcp", server.Addr().String())
	c.Assert(err, IsNil)

	// Create an IRCConnection from the established connection.
	ep := Endpoint{"some server"}
	ic := FromRWC(conn, []Endpoint{ep})

	c.Check(ic.State(), Equals, CONNECTED)
}
