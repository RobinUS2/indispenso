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
	"strconv"
	"strings"
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
		http.HandleFunc("/meta", metaHandler)
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
	if len(body) > 0 {
		if debug {
			log.Println(fmt.Sprintf("REQ BODY %s", body))
		}

		// Parse response
		b = []byte(body)
		var f interface{}
		err := json.Unmarshal(b, &f)
		if err != nil {
			log.Println(fmt.Sprintf("ERR: Failed to parse request body json %s"), err)
		} else {
			var port int
			bodyData := f.(map[string]interface{})
			// Node discovery?
			if bodyData["nodes"] != nil {
				var nodes []string = strings.Split(fmt.Sprintf("%s", bodyData["nodes"]), ",")
				for _, node := range nodes {
					// Skip empty
					node = strings.TrimSpace(node)
					if len(node) == 0 {
						continue
					}
					nodeSplit := strings.Split(node, ":")
					var err error
					port, err = strconv.Atoi(nodeSplit[1])
					if err != nil {
						log.Println(fmt.Sprintf("ERROR: Discovered %s port format invalid", node))
						continue
					}
					n := discoveryService.NewNode(nodeSplit[0], port, getPulicIp(nodeSplit[0]))
					if discoveryService.AddNode(n) {
						// Broadcast cluster join
						discoveryService.NotifyJoin()
					}
				}
			}
		}
	}
}

// Meta handler
func metaHandler(w http.ResponseWriter, r *http.Request) {
	// Log request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to read request body %s"), err)
		return
	}
	if len(body) > 0 {
		if debug {
			log.Println(fmt.Sprintf("REQ BODY %s", body))
		}
	}
}

// Task handler
func taskHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

// Config handler
func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}
