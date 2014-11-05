package ircconnection

import (
	. "gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type IRCConnectionTest struct{}

var _ = Suite(&IRCConnectionTest{})

func (s *IRCConnectionTest) TestNewIRCConnection(c *C) {
	ep := Endpoint{"some server"}
	conn := NewIRCConnection([]Endpoint{ep})

	c.Check(conn.State(), Equals, DISCONNECTED)
	c.Assert(len(conn.Endpoints), Equals, 1)
	c.Check(conn.Endpoints[0], Equals, ep)
}
