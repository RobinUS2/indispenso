package main

import (
	"flag"
	"log"
)

// @author Robin Verlangen
// Indispenso: Distribute, manage, regulate, arrange. Simple & secure management based on consensus.

var conf *Conf
var isServer bool
var isClient bool
var seedUri string
var server *Server
var client *Client

func main() {
	conf = newConf()

	// Read flags
	flag.BoolVar(&isServer, "server", false, "Should this run the server process")
	flag.StringVar(&seedUri, "seed", "", "Seed URI")
	flag.Parse()
	log.Printf("Server %t", isServer)
	log.Printf("Client %t", isClient)
	log.Printf("Seed %s", seedUri)
	isClient = len(seedUri) > 0

	// Server
	if isServer {
		server = newServer()
		server.Start()
	}

	// Client
	if isClient {
		server = newServer()
		server.Start()
	}
}