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
	"time"
	"sync"
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
var msgLog map[string]bool = make (map[string]bool )
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

		// Calculate & validate message digest
		mac := hmac.New(sha256.New, secretKey)
		mac.Write(b)
		expectedSignature := mac.Sum(nil)
		headerSigVal := r.Header.Get("X-Message-Digest")
		headerSigValSplit := strings.Split(headerSigVal, "sha256=")
		headerSigHexDec, errHex := hex.DecodeString(headerSigValSplit[1])
		if errHex != nil {
			log.Println(fmt.Sprintf("ERR: Failed to decode hex digest %s"), errHex)
			return
		}
		signature := []byte(headerSigHexDec)
		if hmac.Equal(expectedSignature, signature) == false {
			log.Println(fmt.Sprintf("ERR: Message digest header invalid, dropping message"))
			log.Println(fmt.Sprintf("%b", expectedSignature))
			log.Println(fmt.Sprintf("%b", signature))
			return
		}

		// Parse response
		var f interface{}
		err := json.Unmarshal(b, &f)
		if err != nil {
			log.Println(fmt.Sprintf("ERR: Failed to parse request body json %s"), err)
		} else {
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
}
