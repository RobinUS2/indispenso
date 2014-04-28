// @author Robin Verlangen
// HTTP server used for communication

package main

// Imports
import (
	"log"
	"net/http"
	"fmt"
)

// Server
type Server struct {

}

// Init server
func NewServer() *Server {
	return &Server{}
}

// Start server
func (s *Server) Start() bool {
	log.Println("INFO: Starting server")
	go func() {
		http.HandleFunc("/", handler)

	}()
	return true
}

// HTTP handler
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}