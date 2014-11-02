// The ircsession package provides a state machine interface to an IRC session.
package ircsession

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/msparks/iq/ircconnection"
	"github.com/msparks/iq/notify"
	"github.com/sorcix/irc"
	ircproto "github.com/msparks/iq/public/irc"
	"log"
	"sync"
	"time"
)

type State string

const (
	DISCONNECTED State = "DISCONNECTED"
	CONNECTING   State = "CONNECTING"
	HANDSHAKING  State = "HANDSHAKING"
	CONNECTED    State = "CONNECTED"
)

type IRCSettings struct {
	Nicknames []string
	User string
	Realname string
}

type IRCSession struct {
	notify.Notifier

	Conn *ircconnection.IRCConnection

	// IRC protocol parameters.
	settings IRCSettings

	state State
	mu sync.Mutex
}

func NewIRCSession(settings IRCSettings, conn *ircconnection.IRCConnection) *IRCSession {
	// TODO(msparks): handle connections not in DISCONNECTED state.
	s := &IRCSession{
		Conn: conn,
		settings: settings,
		state: DISCONNECTED,
	}
	go s.run()
	return s
}

func (s *IRCSession) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *IRCSession) run() {
	notifiee := s.Conn.NewNotifiee()
	defer s.Conn.CloseNotifiee(notifiee)

	for {
		v := <-notifiee
		log.Printf("IRCSession: notification %T", v)

		switch v := v.(type) {
		case ircconnection.StateChangeNotification:
			switch s.Conn.State() {
			case ircconnection.DISCONNECTED:
				s.state = DISCONNECTED
				time.Sleep(5 * time.Second)
				log.Printf("Reconnecting...")
				s.state = CONNECTING
				go s.Conn.StateIs(ircconnection.CONNECTING)

			case ircconnection.CONNECTED:
				s.state = HANDSHAKING
				nick := &ircproto.Message{
					Type: ircproto.Message_NICK.Enum(),
					Nick: &ircproto.Nick{
						NewNick: proto.String(s.settings.Nicknames[0]),
					},
				}
				user := &ircproto.Message{
					Type: ircproto.Message_USER.Enum(),
					User: &ircproto.User{
						User: proto.String(s.settings.User),
						Realname: proto.String(s.settings.Realname),
					},
				}
				s.Conn.OutgoingMessageIs(nick)
				s.Conn.OutgoingMessageIs(user)
			}

		case ircconnection.IncomingMessageNotification:
			switch v.Message.GetType() {
			case ircproto.Message_PING:
				s.onPing(v.Message)

			case ircproto.Message_REPLY:
				r := v.Message.GetReply()
				if r.GetNumeric() == irc.RPL_WELCOME {
					s.onWelcome(r)
				}
			}
		}
	}
}

func (s *IRCSession) onPing(msg *ircproto.Message) {
	target := msg.GetPing().GetTarget()
	reply := &ircproto.Message{
		Type: ircproto.Message_PONG.Enum(),
		Pong: &ircproto.Pong{
			Target: proto.String(target),
		},
	}
	s.Conn.OutgoingMessageIs(reply)
}

func (s *IRCSession) onWelcome(m *ircproto.Reply) {
	params := m.GetParams()
	if len(params) > 0 {
		s.state = CONNECTED
		log.Printf("Connected. Nick is %s", params[0])
	}
}
