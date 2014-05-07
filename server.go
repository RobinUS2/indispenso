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
	"github.com/gorilla/sessions"
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

// Cookie session store
var store = sessions.NewCookieStore([]byte("hndvsvnhgihn1rseil8xghveiu"))

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
		http.HandleFunc("/data-repair", dataRepairHandler)
		http.HandleFunc("/app", appHandler)
		http.HandleFunc("/api", apiHandler)
		http.Handle("/app/static/", http.StripPrefix("/app/static/", http.FileServer(http.Dir(APP_STATIC_PATH))))
		if testMode {
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
	if trace && body != nil && len(body) > 0 {
		log.Println(fmt.Sprintf("TRACE: Request body %s", body))
	}

	// To byte array
	b := []byte(body)

	// Calculate & validate message digest
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(b)
	expectedSignature := mac.Sum(nil)
	headerSigVal := r.Header.Get("X-Message-Digest")
	headerSigValSplit := strings.Split(headerSigVal, "sha256=")
	if len(headerSigValSplit) != 2 {
		return nil, newErr(fmt.Sprintf("Failed to read message digest header"))
	}
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

	// Parse json
	var f interface{}
	err = json.Unmarshal(b, &f)
	if err != nil {
		return nil, newErr(fmt.Sprintf("Failed to parse request json, unable to parse message content: %s", err))
	}
	jsonData := f.(map[string]interface{})

	// Check timestamp for age
	if jsonData["ts"] == nil {
		return nil, newErr(fmt.Sprintf("Missing message timestamp"))
	}
	nowNano := time.Now().UnixNano()
	minTs := nowNano - (3 * 1000000000) // Maximum of 3 seconds
	msgTs, tsErr := strconv.ParseInt(fmt.Sprintf("%s", jsonData["ts"]), 10, 64)
	if tsErr != nil {
		return nil, newErr(fmt.Sprintf("ERR: Invalid message timestamp %s", tsErr))
	}
	if msgTs < minTs {
		return nil, newErr(fmt.Sprintf("Message timestamp too old, dropping to prevent replay attack"))
	}

	// Check if this message id has been used before (to prevent replay)
	if jsonData["msg_id"] == nil {
		return nil, newErr(fmt.Sprintf("Missing message id"))
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

// Data repair handler
func dataRepairHandler(w http.ResponseWriter, r *http.Request) {
	// Read and validate request
	_, err := readRequest(w, r)
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

	// Output datastore memtable as JSON
	fmt.Fprintf(w, datastore.memTableToJson())
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

		// Already replicated?
		var replicated bool = false
		if bodyData["r"] != nil {
			// This is a replication message from another node, do not replicate again
			replicated = true
		}

		// Create mutation object
		m := datastore.CreateMutation()
		m.Key = fmt.Sprintf("%s", bodyData["k"])
		m.Value = fmt.Sprintf("%s", bodyData["v"])
		ts, tsErr := strconv.ParseInt(fmt.Sprintf("%s", bodyData["ts"]), 10, 64)
		if tsErr != nil {
			log.Println(fmt.Sprintf("ERR: Invalid timestamp %s", tsErr))
			return
		}
		m.Timestamp = ts
		m.Replicated = replicated

		// Mutation mode
		if bodyData["m"] != nil {
			modeStr := fmt.Sprintf("%s", bodyData["m"])
			if len(modeStr) > 0 {
				mode, modeErr := strconv.ParseInt(modeStr, 10, 0)
				if modeErr != nil {
					log.Println(fmt.Sprintf("ERR: Invalid mutation mode %s", modeErr))
					return
				}
				if mode > 0 {
					m.MutationMode = int(mode)
				}
			}
		}

		// Push mutation into datastore
		datastore.PushMutation(m)

		// Respond
		fmt.Fprintf(w, "{\"ok\":true}")
	} else {
		w.WriteHeader(400)
	}
}

// Web application handler
func appHandler(w http.ResponseWriter, r *http.Request) {
	// Read application html
	appHtmlBytes, appHtmlBytesErr := ioutil.ReadFile(APP_HTML_FILE)
	if appHtmlBytesErr != nil {
		log.Fatal("ERR: Failed to read application interface")
	}
	appHtml = string(appHtmlBytes)
	fmt.Fprintf(w, appHtml)
}

// API handler
func apiHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	method := params.Get("method")
	jsonStr := params.Get("json")

	// Read body
	if len(jsonStr) == 0 {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(fmt.Sprintf("ERR: Failed to read request body %s", err))
		} else {
			jsonStr = string(body)
		}
	}

	// Json header
	w.Header().Set("Content-Type", "application/json")

	// Parse json
	jsonData := api.parseJson(jsonStr)

	// setup session store
	session, _ := store.Get(r, "auth-session")

	// Authenticate user
	var user *User = nil
	if method != "auth" {
		if api.checkSession(jsonData) == false && api.checkSessionString(session.Values["session_token"].(string)) == false {
			// Not authenticated
			log.Println(fmt.Sprintf("WARN: User not authenticated"))
			w.WriteHeader(401)
			return
		}
		user = api.getUser(jsonData)
	}

	// Handle methods
	var respData map[string]interface{}
	if testMode && method == "mirror" {
		// Test method
		respData = api.Mirror(jsonData)
	} else if method == "auth" {
		// Authenticate user
		respData = api.Auth(jsonData)
		if respData["session_token"] != nil {
			// We are authenticated
			// Save to session cookie so we can read it on next request

			session.Values["session_token"] = respData["session_token"]
			// Set default expiration
			session.Options = &sessions.Options{
				Path:   "/",
				MaxAge: 1800, // 30 minutes
			}
			// Save it.
			session.Save(r, w)
		}
	} else if method == "custom_command" {
		// Custom command
		if user.IsAdmin == false {
			log.Println(fmt.Sprintf("WARN: Admins only"))
			w.WriteHeader(403)
			return
		}
		respData = api.CustomCommand(jsonData)
	} else {
		// Not supported
		w.WriteHeader(400)
	}

	// To json
	b, err := json.Marshal(respData)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to format json %s", err))
		return
	}
	resp := string(b)

	fmt.Fprintf(w, resp)
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
		mutation["m"] = params.Get("m")
		resp, _ := discoveryService.Nodes[0].sendData("data", msgToJson(mutation))
		fmt.Fprintf(w, resp)

	} else if method == "list_data" {
		// List all data in json format
		// @example http://localhost:8011/test?method=list_data
		if datastore == nil {
			log.Println(fmt.Sprintf("ERR: Datastore not yet started"))
			w.WriteHeader(503)
			return
		}
		fmt.Fprintf(w, datastore.memTableToJson())

	} else if method == "data_get" {
		// Get data for a key
		// @example http://localhost:8011/test?method=data_get&k=my_key
		if datastore == nil {
			log.Println(fmt.Sprintf("ERR: Datastore not yet started"))
			w.WriteHeader(503)
			return
		}

		e, _ := datastore.GetEntry(params.Get("k"))
		if e != nil {
			fmt.Fprintf(w, e.Value)
		}
	} else {
		// Not supported
		w.WriteHeader(400)
	}
}
