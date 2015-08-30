package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// @author Robin Verlangen
// Indispenso: Distribute, manage, regulate, arrange. Simple & secure management based on consensus.

var conf *Conf
var serverPort int
var isClient bool
var clientPort int
var seedUri string
var server *Server
var client *Client
var log *Log
var hostname string
var hostnameOverride string
var debug bool
var autoTag bool
var shutdown chan bool = make(chan bool)

const CLIENT_PING_INTERVAL int = 60                       // In seconds
const LONG_POLL_TIMEOUT time.Duration = time.Duration(30) // In seconds
const DEFAULT_COMMAND_TIMEOUT int = 300                   // In seconds

func main() {
	log.Println("Starting indispenso")

	// Log
	log = newLog()

	// Conf
	conf = newConf()
	conf.load()
	conf.startAutoReload()

	// Read flags
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.BoolVar(&autoTag, "auto-tag", true, "Auto tag based on server details")
	flag.StringVar(&seedUri, "seed", "", "Seed URI")
	flag.StringVar(&hostnameOverride, "hostname", "", "Hostname")
	flag.IntVar(&serverPort, "server-port", 897, "Server port")
	flag.IntVar(&clientPort, "client-port", 898, "Client port")
	flag.Parse()

	// Hostname
	if len(hostnameOverride) < 1 {
		hostname, _ = os.Hostname()
		hostname = strings.ToLower(hostname)
	} else {
		hostname = hostnameOverride
	}
	log.Printf("Hostname %s", hostname)

	// Auto tag
	if autoTag {
		conf.autoTag()
	}

	// Seed override?
	if len(seedUri) > 0 {
		conf.Seed = seedUri
	} else {
		seedUri = conf.Seed
	}

	// Must have token
	minLen := 32
	if len(strings.TrimSpace(conf.SecureToken)) < minLen {
		log.Fatal(fmt.Sprintf("Must have secure token with minimum length of %d", minLen))
	}

	// Server
	if conf.IsServer {
		server = newServer()
		server.Start()

		// Empty seed? Then go for local
		if len(seedUri) < 1 {
			seedUri = fmt.Sprintf("https://127.0.0.1:%d/", serverPort)
			conf.Seed = seedUri

			// Sleep for 1 second to allow the server to start
			time.Sleep(1 * time.Second)
		}
	}

	// Client
	isClient = len(seedUri) > 0
	if isClient {
		client = newClient()
		client.Start()
	}

	// Wait for shutdown
	<-shutdown
}
