package main

import "errors"
import "io"
import "log"
import "sync"
import "strings"
import "time"
import "code.google.com/p/gcfg"
import "code.google.com/p/goprotobuf/proto"
import "github.com/msparks/iq/public"
import ircproto "github.com/msparks/iq/public/irc"
import "github.com/sorcix/irc"
import "math/rand"
import "net"
import "net/http"
import "net/rpc"
import "net/rpc/jsonrpc"
import "strconv"

type Config struct {
	Network map[string]*NetworkConfig
	Channel map[string]*ChannelConfig
}

type NetworkConfig struct {
	Nick   string
	Server string
	// TODO(msparks): Multiple servers on the same network?
}

type ChannelConfig struct {
	Label []string
}

type Network struct {
	Name     string
	Channels []*Channel
	Config   *NetworkConfig
}

type Channel struct {
	Name   string
	Config *ChannelConfig
}

type ConnectionHandle string

type NetworkConnection struct {
	Network *Network
	Conn *irc.Conn
	Handle ConnectionHandle
}

func writeMessage(nc *NetworkConnection, m *irc.Message) {
	log.Printf("[%s] >> %v", nc.Network.Name, m)
	err := nc.Conn.Encode(m)
	if err != nil {
		log.Fatal(err)
	}
}

func runNetworkLoop(net *Network, cfg *Config, evs *EventServer) {
	log.Printf("Network %s: config=%+v", net.Name, *net.Config)

	for {
		nc := NetworkConnection{Network: net}
		var err error
		for {
			nc.Handle = ConnectionHandle(strconv.FormatInt(rand.Int63(), 16))
			log.Printf("[%s] Connecting to %s.", net.Name, net.Config.Server)
			nc.Conn, err = irc.Dial(net.Config.Server)
			if err != nil {
				log.Printf(
					"[%s] Connection error: %v. Retrying in 5 seconds.",
					net.Name, err)
				time.Sleep(5 * time.Second)
			} else {
				break
			}
		}

		log.Printf("[%s] Connected to %s. Handle: %s", net.Name, net.Config.Server, nc.Handle)
		runNetworkConnection(nc, cfg, evs)
		log.Printf("[%s] Disconnected from %s.", net.Name, net.Config.Server)
		time.Sleep(10 * time.Second)
	}
}

func runNetworkConnection(nc NetworkConnection, cfg *Config, evs *EventServer) {
	var authed bool
	for {
		message, err := nc.Conn.Decode()
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

			writeMessage(&nc, nick)
			writeMessage(&nc, user)

			authed = true
		}

		if message.Command == irc.RPL_WELCOME {
			// We're connected. Join configured channels.
			for _, channel := range nc.Network.Channels {
				join := &irc.Message{nil, irc.JOIN, []string{channel.Name}, ""}
				writeMessage(&nc, join)
			}
		}

		p, err := ircProtoMessage(message)
		if err != nil {
			continue
		}
		ev := &public.Event{IrcMessage: &public.IrcMessage{
			Handle: proto.String(string(nc.Handle)),
			Message: p}}
		evs.Event <- ev

		if p.GetType() == ircproto.Message_PING {
			ping := p.GetPing()
			var params []string
			if ping.GetSource() != "" {
				params = append(params, ping.GetSource())
			}
			pong := &irc.Message{nil, irc.PONG, params, ping.GetTarget()}
			writeMessage(&nc, pong)
		}
	}
}

func prefixProto(prefix *irc.Prefix) (p *ircproto.Prefix) {
	p = &ircproto.Prefix{
		Name: proto.String(prefix.Name),
		User: proto.String(prefix.User),
		Host: proto.String(prefix.Host),
	}
	return p
}

func ircProtoMessage(message *irc.Message) (p *ircproto.Message, err error) {
	p = &ircproto.Message{
		Type: ircproto.Message_UNKNOWN.Enum(),
	}

	switch message.Command {
	case irc.PING:
		var source string
		if len(message.Params) > 0 {
			source = message.Params[0]
		}
		p.Type = ircproto.Message_PING.Enum()
		p.Ping = &ircproto.Ping{
			Source: proto.String(source),
			Target: proto.String(message.Trailing),
		}

	case irc.PRIVMSG:
		var target string
		if len(message.Params) > 0 {
			target = message.Params[0]
		}
		p.Type = ircproto.Message_PRIVMSG.Enum()
		p.Privmsg = &ircproto.Privmsg{
			Source:  prefixProto(message.Prefix),
			Target:  proto.String(target),
			Message: proto.String(message.Trailing),
		}

	case irc.NOTICE:
		var target string
		if len(message.Params) > 0 {
			target = message.Params[0]
		}
		p.Type = ircproto.Message_NOTICE.Enum()
		p.Notice = &ircproto.Notice{
			Source:  prefixProto(message.Prefix),
			Target:  proto.String(target),
			Message: proto.String(message.Trailing),
		}

	default:
		return nil, errors.New("Unknown command")
	}

	return p, nil
}

func startCommandRpcServer() {
	cmd := new(CmdServer)

	s := rpc.NewServer()
	s.Register(cmd)
	s.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	l, e := net.Listen("tcp", "[::]:8222")
	if e != nil {
		log.Fatal("Command server listen error: ", e)
	}

	log.Print("Command server listening.")

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go s.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "IQ\n")
}

func startStreamServer(s *EventServer) {
	http.HandleFunc("/", handleIndex)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWebsocket(s, w, r)
	})

	log.Print("Starting stream server.")

	err := http.ListenAndServe("[::]:8223", nil)
	if err != nil {
		log.Fatal("Stream server error: ", err)
	}
}

func readConfig(filename string) (cfg Config, err error) {
	err = gcfg.ReadFileInto(&cfg, filename)
	return cfg, err
}

func main() {
	log.Print("Starting.")

	// Find and read configuration file.
	cfg, err := readConfig("iq.conf")
	if err != nil {
		log.Fatal(err)
	}

	networks := make(map[string]*Network)

	for name, config := range cfg.Network {
		log.Printf("Network (%s): %v", name, config)
		networks[name] = &Network{name, nil, config}
	}

	for name, config := range cfg.Channel {
		fields := strings.FieldsFunc(name, func(c rune) bool { return c == ',' })
		if len(fields) != 2 {
			log.Fatalf("Channel section name must be '<network>,<channel>'. Got: %s",
				name)
		}
		netName, channelName := fields[0], fields[1]

		if networks[netName] == nil {
			log.Fatalf("Unknown network '%s' for channel '%s'.", netName, channelName)
		}

		log.Printf("Channel %s on network %s: %v", channelName, netName, config)
		networks[netName].Channels = append(
			networks[netName].Channels, &Channel{channelName, config})
	}

	eventServer := NewEventServer()

	// Start RPC server.
	go startCommandRpcServer()

	// Stream server.
	go startStreamServer(eventServer)

	// Connect to configured networks.
	var wg sync.WaitGroup
	for _, network := range networks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runNetworkLoop(network, &cfg, eventServer)
		}()
		time.Sleep(10 * time.Second)
	}

	wg.Wait()
	log.Print("Exiting.")
}
