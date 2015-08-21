package main

import (
	"strings"
	"flag"
	"fmt"
	"os"
	"time"
)

// @author Robin Verlangen
// Indispenso: Distribute, manage, regulate, arrange. Simple & secure management based on consensus.

var conf *Conf
var isServer bool
var serverPort int
var isClient bool
var clientPort int
var seedUri string
var server *Server
var client *Client
var log *Log
var hostname string
var debug bool
var secureToken string
var shutdown chan bool = make(chan bool)
const CLIENT_PING_INTERVAL int = 60 // In seconds
const LONG_POLL_TIMEOUT time.Duration = time.Duration(30) // In seconds
const DEFAULT_COMMAND_TIMEOUT int = 60 // In seconds

func main() {
	// Log
	log = newLog()

	// Conf
	conf = newConf()

	// Read flags
	flag.BoolVar(&isServer, "server", false, "Should this run the server process")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.StringVar(&seedUri, "seed", "", "Seed URI")
	flag.StringVar(&secureToken, "secure-token", "", "Secure token")
	flag.IntVar(&serverPort, "server-port", 897, "Server port")
	flag.IntVar(&clientPort, "client-port", 898, "Client port")
	flag.Parse()

	// Must have token
	minLen := 32
	if len(strings.TrimSpace(secureToken)) < minLen {
		log.Fatal(fmt.Sprintf("Must have secure token with minimum length of %d", minLen))
	}

	// Hostname
	hostname, _ = os.Hostname()

	// Server
	if isServer {
		server = newServer()
		server.Start()

		// Empty seed? Then go for local
		if len(seedUri) < 1 {
			seedUri = fmt.Sprintf("https://127.0.0.1:%d/", serverPort)

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
	<- shutdown
}