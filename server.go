// @author Robin Verlangen
// HTTP server used for communication

package main

// Imports
import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Server
type Server struct {
}

// Init server
func NewServer() *Server {
	return &Server{}
}

// Message id map (for replay protection)
// @todo Improve as we do not have to keep everything in here forever, is memory leaking basically
var msgLog map[string]bool = make(map[string]bool)
var msgLogMux sync.RWMutex

// Is this message seen before?
func isMsgSeen(msgId string) bool {
	msgLogMux.RLock()
	defer msgLogMux.RUnlock()
	return msgLog[msgId]
}

// Mark message as seen
func markMsgSeen(msgId string) bool {
	msgLogMux.Lock()
	msgLog[msgId] = true
	msgLogMux.Unlock()
	return true
}

// Start server
func (s *Server) Start() bool {
	log.Println(fmt.Sprintf("INFO: Starting server on port %d", serverPort))
	go func() {
		http.HandleFunc("/discovery", discoveryHandler)
		http.HandleFunc("/meta", metaHandler)
		http.HandleFunc("/data", dataHandler)
		if testing {
			http.HandleFunc("/test", testHandler)
		}
		http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil)
	}()
	return true
}

// Validate request
func readRequest(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	// Log request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, newErr(fmt.Sprintf("Failed to read request body %s", err))
	}
	//if len(body) > 0 {
	if debug && body != nil && len(body) > 0 {
		log.Println(fmt.Sprintf("DEBUG: Request body %s", body))
	}

	// To byte array
	b := []byte(body)

	// Calculate & validate message digest
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(b)
	expectedSignature := mac.Sum(nil)
	headerSigVal := r.Header.Get("X-Message-Digest")
	headerSigValSplit := strings.Split(headerSigVal, "sha256=")
	headerSigHexDec, errHex := hex.DecodeString(headerSigValSplit[1])
	if errHex != nil {
		return nil, newErr(fmt.Sprintf("Failed to decode hex digest %s", errHex))
	}
	signature := []byte(headerSigHexDec)
	if hmac.Equal(expectedSignature, signature) == false {
		log.Println(fmt.Sprintf("ERR: Message digest header invalid, dropping message"))
		log.Println(fmt.Sprintf("%b", expectedSignature))
		log.Println(fmt.Sprintf("%b", signature))
		return nil, newErr("Message digest header invalid")
	}

	// Check if this message id has been used before (to prevent replay)
	var f interface{}
	err = json.Unmarshal(b, &f)
	if err != nil {
		return nil, newErr(fmt.Sprintf("Failed to parse request json, unable to validate message id: %s", err))
	}
	jsonData := f.(map[string]interface{})
	if jsonData["msg_id"] == nil {
		return nil, newErr(fmt.Sprintf("Missing message id: %s", err))
	}

	// Seen?
	msgId := strings.TrimSpace(fmt.Sprintf("%s", jsonData["msg_id"]))
	if len(msgId) == 0 {
		return nil, newErr(fmt.Sprintf("Missing message id content: %s", err))
	}
	if isMsgSeen(msgId) {
		return nil, newErr(fmt.Sprintf("Message id %s already seen, dropping to prevent replay attack", msgId))
	}
	markMsgSeen(msgId)

	// OK :)
	return b, nil
}

// Discovery handler
func discoveryHandler(w http.ResponseWriter, r *http.Request) {
	// Read and validate request
	b, err := readRequest(w, r)
	if err != nil {
		// No log, is already written
		return
	}

	// Parse response
	if len(b) > 0 {
		var f interface{}
		err = json.Unmarshal(b, &f)
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
						log.Println(fmt.Sprintf("ERR: Discovered %s port format invalid", node))
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

	// Current time
	var now time.Time = time.Now().UTC()

	// Data
	var data map[string]string = make(map[string]string)
	data["time"] = fmt.Sprintf("%s", now)

	// To JSON
	b, err = json.Marshal(data)
	if err == nil {
		fmt.Fprint(w, fmt.Sprintf("%s", b))
	} else {
		fmt.Fprint(w, "Failed to format json")
	}
}

// Meta handler
func metaHandler(w http.ResponseWriter, r *http.Request) {
	// Read and validate request
	b, err := readRequest(w, r)
	if err != nil {
		// No log, is already written
		return
	}

	// Parse response
	if len(b) > 0 {
		// Parse response
		var f interface{}
		err := json.Unmarshal(b, &f)
		if err != nil {
			log.Println(fmt.Sprintf("ERR: Failed to parse request body json %s"), err)
			return
		}

		bodyData := f.(map[string]interface{})
		// Basic validation of type
		if bodyData["type"] == nil || len(fmt.Sprintf("%s", bodyData["type"])) == 0 {
			log.Println(fmt.Sprintf("ERR: Missing type"))
			return
		}
		metaType := fmt.Sprintf("%s", bodyData["type"])

		// Basic send validation
		if bodyData["sender"] == nil || len(fmt.Sprintf("%s", bodyData["sender"])) == 0 {
			log.Println(fmt.Sprintf("ERR: Missing sender"))
			return
		}
		metaSender := fmt.Sprintf("%s", bodyData["sender"])
		metaSenderPort, _ := strconv.Atoi(fmt.Sprintf("%s", bodyData["sender_port"]))

		// Execute action
		if metaType == "node_leave" {
			// Node leaving
			if discoveryService != nil {
				for _, n := range discoveryService.Nodes {
					// Check proper host
					if n.Host == metaSender && n.Port == metaSenderPort {
						if discoveryService.RemoveNode(n) {
							// Done
							break
						}
					}
				}
			}
		}

	}
}

// Data handler (storage of key values, with conflict resolution and eventual consistency)
func dataHandler(w http.ResponseWriter, r *http.Request) {
	// Read and validate request
	b, err := readRequest(w, r)
	if err != nil {
		// No log, is already written
		return
	}

	// Datastore?
	if datastore == nil {
		log.Println(fmt.Sprintf("ERR: Datastore not yet started"))
		w.WriteHeader(503)
		return
	}

	// Parse response
	if len(b) > 0 {
		// Parse response
		var f interface{}
		err := json.Unmarshal(b, &f)
		if err != nil {
			log.Println(fmt.Sprintf("ERR: Failed to parse request body json %s"), err)
			return
		}
		bodyData := f.(map[string]interface{})

		// Key?
		if bodyData["k"] == nil {
			log.Println("ERR: Missing key")
			return
		}

		// Value
		if bodyData["v"] == nil {
			log.Println("ERR: Missing value")
			return
		}

		// Submit mutation
		m := datastore.CreateMutation()
		m.key = fmt.Sprintf("%s", bodyData["k"])
		m.value = fmt.Sprintf("%s", bodyData["v"])
		ts, tsErr := strconv.ParseInt(fmt.Sprintf("%s", bodyData["ts"]), 10, 64)
		if tsErr != nil {
			log.Println("ERR: Invalid timestamp %s", tsErr)
			return
		}
		m.timestamp = ts

		// Respond
		fmt.Fprintf(w, "{\"ok\":true}")
	} else {
		w.WriteHeader(400)
	}
}

// Test handler
func testHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	method := params.Get("method")

	// Methods
	if method == "list_nodes" {
		// List nodes
		// @example http://localhost:8011/test?method=list_nodes
		if discoveryService == nil {
			log.Println(fmt.Sprintf("ERR: Discovery service not yet started"))
			w.WriteHeader(503)
			return
		}
		var data []string = make([]string, 0)
		for _, n := range discoveryService.Nodes {
			data = append(data, n.FullName())
		}
		b, err := json.Marshal(data)
		if err != nil {
			log.Println(fmt.Sprintf("ERR: Failed to format json"))
			return
		}
		fmt.Fprintf(w, fmt.Sprintf("%s", b))
	} else if method == "data_mutate" {
		// Mutate a key
		// @example http://localhost:8011/test?method=data_mutate&k=my_key&v=my_value
		mutation := getEmptyMetaMsg("data_mutation")
		mutation["k"] = params.Get("k")
		mutation["v"] = params.Get("v")
		resp, _ := discoveryService.Nodes[0].sendData("data", msgToJson(mutation))
		fmt.Fprintf(w, resp)
	} else {
		// Not supported
		w.WriteHeader(400)
	}
}
