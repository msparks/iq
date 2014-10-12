package main

import "log"
import "sync"
import "strings"
import "time"
import "code.google.com/p/gcfg"
import "github.com/sorcix/irc"

type Config struct {
	Network map[string]*NetworkConfig
	Channel map[string]*ChannelConfig
}

type NetworkConfig struct {
	Nick   string
	Server string
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

func writeMessage(net *Network, c *irc.Conn, m *irc.Message) {
	log.Printf("[%s] >> %v", net.Name, m)
	err := c.Encode(m)
	if err != nil {
		log.Fatal(err)
	}
}

func runNetworkLoop(net *Network, cfg *Config) {
	log.Printf("Network %s: config=%+v", net.Name, *net.Config)

	var c *irc.Conn
	var err error
	for {
		for {
			log.Printf("[%s] Connecting to %s.", net.Name, net.Config.Server)
			c, err = irc.Dial(net.Config.Server)
			if err != nil {
				log.Printf(
					"[%s] Connection error: %v. Retrying in 5 seconds.",
					net.Name, err)
				time.Sleep(5 * time.Second)
			} else {
				break
			}
		}

		log.Printf("[%s] Connected to %s.", net.Name, net.Config.Server)
		runNetworkConnection(net, c, cfg)
		log.Printf("[%s] Disconnected from %s.", net.Name, net.Config.Server)
		time.Sleep(10 * time.Second)
	}
}

func runNetworkConnection(net *Network, c *irc.Conn, cfg *Config) {
	var authed bool
	for {
		message, err := c.Decode()
		if err != nil {
			return
		}

		log.Printf(
			"[%s] %v %v %v %v",
			net.Name,
			message.Prefix,
			message.Command,
			message.Params,
			message.Trailing)

		if !authed {
			nick := &irc.Message{nil, irc.NICK, []string{net.Config.Nick}, ""}
			user := &irc.Message{nil, irc.USER, []string{"iq", "*", "*"}, "IQ"}

			writeMessage(net, c, nick)
			writeMessage(net, c, user)

			authed = true
		}

		if message.Command == irc.PING {
			pong := &irc.Message{nil, irc.PONG, nil, message.Trailing}
			writeMessage(net, c, pong)
		}

		if message.Command == irc.RPL_WELCOME {
			// We're connected. Join configured channels.
			for _, channel := range net.Channels {
				join := &irc.Message{nil, irc.JOIN, []string{channel.Name}, ""}
				writeMessage(net, c, join)
			}
		}
	}
}

func readConfig(filename string) (cfg Config, err error) {
	err = gcfg.ReadFileInto(&cfg, filename)
	return cfg, err
}

func main() {
	log.Print("Starting.")

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

	var wg sync.WaitGroup
	for _, network := range networks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runNetworkLoop(network, &cfg)
		}()
		time.Sleep(10 * time.Second)
	}

	wg.Wait()
	log.Print("Exiting.")
}
