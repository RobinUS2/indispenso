package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

// @author Robin Verlangen
// Indispenso: Distribute, manage, regulate, arrange. Simple & secure management based on consensus.

var conf *Conf
var server *Server
var client *Client
var log *Log
var debug bool
var shutdown chan bool = make(chan bool)

const CLIENT_PING_INTERVAL int = 60                       // In seconds
const LONG_POLL_TIMEOUT time.Duration = time.Duration(30) // In seconds
const DEFAULT_COMMAND_TIMEOUT int = 300                   // In seconds

func main() {
	// Log
	log = newLog()
	//Conf
	conf = newConfig()
	if conf.IsHelp() {
		conf.PrintHelp()
	}

	conf.EnableAutoUpdate()

	log.Println("Starting indispenso")
	if err := conf.Validate(); err != nil {
		log.Fatal(err)
	}

	// Handle signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Printf("Shutting down %s", conf.Hostname)
			os.Exit(0)
		}
	}()

	if conf.ServerEnabled {
		server = newServer()
		server.Start()

		// Empty seed? Then go for local
		if len(conf.EndpointURI) < 1 {
			conf.EndpointURI = fmt.Sprintf("https://127.0.0.1:%d/", conf.ServerPort)
			// Sleep for 1 second to allow the server to start
			time.Sleep(1 * time.Second)
		}
	}

	// Client
	if conf.isClientEnabled() {
		client = newClient()
		client.Start()
	}

	// Wait for shutdown
	<-shutdown
}
