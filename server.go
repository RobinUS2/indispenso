// @author Robin Verlangen
// HTTP server used for communication

package main

// Imports
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
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
	// Current time
	var now time.Time = time.Now().UTC()

	// Data
	var data map[string]string = make(map[string]string)
	data["time"] = fmt.Sprintf("%s", now)

	// To JSON
	b, err := json.Marshal(data)
	if err == nil {
		fmt.Fprint(w, fmt.Sprintf("%s", b))
	} else {
		fmt.Fprint(w, "Failed to format json")
	}

	// Log request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to read request body %s"), err)
		return
	}
	log.Println(fmt.Sprintf("REQ BODY %s", body))
}

// Task handler
func taskHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

// Config handler
func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}