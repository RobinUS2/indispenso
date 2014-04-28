// @author Robin Verlangen
// HTTP server used for communication

package main

// Imports
import (
	"fmt"
	"log"
	"net/http"
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
	log.Println(fmt.Sprintf("INFO: Starting server on port %d", serverPort))
	go func() {
		http.HandleFunc("/discovery", discoveryHandler)
		http.HandleFunc("/task", taskHandler)
		http.HandleFunc("/config", configHandler)
		http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil)
	}()
	return true
}

// Discovery handler
func discoveryHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

// Task handler
func taskHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

// Config handler
func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}
