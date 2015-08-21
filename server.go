package main

import (
	"net/http"
	"fmt"
	"github.com/julienschmidt/httprouter"
)

// Server methods (you probably only need one or two in HA failover mode)

type Server struct {

}

// Start server
func (s *Server) Start() bool {
	log.Println("Starting server")

	// Start webserver
	go func() {
		router := httprouter.New()
	    router.GET("/ping", Ping)

	    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverPort), router))
    }()
	return true
}

// Ping
func Ping(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    fmt.Fprint(w, "Pong!\n")
}

// Create new server
func newServer() *Server {
	return &Server{}
}