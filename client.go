package main

import (
	"net/http"
	"fmt"
	"github.com/julienschmidt/httprouter"
)

// Client methods (one per "slave", communicates with the server)

type Client struct {

}

// Start client
func (s *Client) Start() bool {
	log.Println("Starting client")

	// Start webserver
	go func() {
		router := httprouter.New()
	    router.GET("/ping", Ping)

	    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", clientPort), router))
    }()
	return true
}

// Create new client
func newClient() *Client {
	return &Client{}
}