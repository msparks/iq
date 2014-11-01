package main

import "io"
import "log"
import "strings"
import "time"
import "code.google.com/p/gcfg"
import "net"
import "net/http"
import "net/rpc"
import "net/rpc/jsonrpc"

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
	var ncs []*NetworkConnection
	for _, network := range networks {
		nc := NewNetworkConnection(network, eventServer)
		ncs = append(ncs, nc)
		time.Sleep(10 * time.Second)
	}
	for _, nc := range ncs {
		nc.Wait()
	}

	log.Print("Exiting.")
}
