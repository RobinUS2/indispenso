package main

import (
	"flag"
	"fmt"
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
var shutdown chan bool = make(chan bool)

func main() {
	// Log
	log = newLog()

	// Conf
	conf = newConf()

	// Read flags
	flag.BoolVar(&isServer, "server", false, "Should this run the server process")
	flag.StringVar(&seedUri, "seed", "", "Seed URI")
	flag.IntVar(&serverPort, "server-port", 17, "Server port")
	flag.IntVar(&clientPort, "client-port", 18, "Client port")
	flag.Parse()

	// Server
	if isServer {
		server = newServer()
		server.Start()

		// Empty seed? Then go for local
		if len(seedUri) < 1 {
			seedUri = fmt.Sprintf("http://127.0.0.1:%d/", serverPort)
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