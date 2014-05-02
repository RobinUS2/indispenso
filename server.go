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

		// OK :)
		return b,nil
	//}
	//return nil, newErr("Empty request")
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
	// Log request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to read request body %s"), err)
		return
	}
	if len(body) > 0 {
		if debug {
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
