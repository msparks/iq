package main

import "code.google.com/p/gcfg"
import "github.com/msparks/iq/ircconnection"
import "github.com/msparks/iq/ircsession"
import "io"
import "log"
import "math/rand"
import "net/http"
import "strconv"
import "strings"
import "time"

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

type NamedSession struct {
	Handle string
	Network *Network
	Conn *ircconnection.IRCConnection
	Session *ircsession.IRCSession
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

	// Stream server.
	go startStreamServer(eventServer)

	// Create a session for each of the configured networks.
	var sessions []*NamedSession
	for _, network := range networks {
		endpoint := ircconnection.Endpoint{Address: network.Config.Server}
		conn := ircconnection.NewIRCConnection([]ircconnection.Endpoint{endpoint})

		settings := ircsession.IRCSettings{
			Nicknames: []string{network.Config.Nick},
			User: "IQ",
			Realname: "IQ",
		}
		session := ircsession.NewIRCSession(settings, conn)

		ns := &NamedSession{
			Handle: strconv.FormatInt(rand.Int63(), 16),
			Network: network,
			Conn: conn,
			Session: session,
		}

		go ConnReactor(ns, eventServer)
		go CommandReactor(eventServer, ns)

		conn.StateIs(ircconnection.CONNECTING)

		sessions = append(sessions, ns)
		time.Sleep(10 * time.Second)
	}

	// Wait forever.
	log.Printf("Main thread sleeping.")
	for {
		time.Sleep(1 * time.Minute)
	}

	log.Print("Exiting.")
}
