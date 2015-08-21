package main

import (
	"net/http"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/RobinUS2/golang-jresp"
	"sync"
	"time"
	"strings"
	"strconv"
	"os"
)

// Server methods (you probably only need one or two in HA failover mode)

type Server struct {
	clientsMux sync.RWMutex
	clients map[string]*RegisteredClient
}

// Register client
func (s *Server) RegisterClient(hostname string) {
	s.clientsMux.Lock()
	if s.clients[hostname] == nil {
		s.clients[hostname] = newRegisteredClient(hostname)
		log.Printf("Client %s registered", hostname)
	}
	s.clients[hostname].mux.Lock()
	s.clients[hostname].LastPing = time.Now()
	s.clients[hostname].mux.Unlock()
	s.clientsMux.Unlock()
}

func (s *Server) GetClient(hostname string) *RegisteredClient {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	return s.clients[hostname]
}

// Scan for old clients
func (s *Server) CleanupClients() {
	s.clientsMux.Lock()
	for k, client := range s.clients {
		if time.Now().Sub(client.LastPing).Seconds() > float64(CLIENT_PING_INTERVAL*5) {
			// Disconnect
			log.Printf("Client %s disconnected", client.Hostname)
			delete(s.clients, k)
		}
	}
	s.clientsMux.Unlock()
}

type RegisteredClient struct {
	mux sync.RWMutex
	Hostname string
	LastPing time.Time
	Cmds map[string]*Cmd
	CmdChan chan bool
}

// Generate keys
func (s *Server) _prepareTlsKeys() {
	if _, err := os.Stat("./private_key"); os.IsNotExist(err) {
		// No keys, generate
		log.Println("Auto-generating keys for server")
		cmd := newCmd("./generate_key.sh", 60)
		cmd.Execute()
		log.Println("Finished generating keys for server")
	}
}

// Start server
func (s *Server) Start() bool {
	log.Println("Starting server")

	// Start webserver
	go func() {
		router := httprouter.New()
	    router.GET("/ping", Ping)
	    router.GET("/client/:hostname/ping", ClientPing)
	    router.GET("/client/:hostname/cmds", ClientCmds)
	    router.POST("/client/:hostname/cmd", PostClientCmd)

	    // Auto generate key
	    s._prepareTlsKeys()

	    // Start server
	    log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", serverPort), "./public_key", "./private_key", router))
    }()

	// Minutely cleanups etc
    go func() {
	    c := time.Tick(1 * time.Minute)
	    for _ = range c {
	    	server.CleanupClients()
	    }
    }()

	return true
}

// Submit client command
func PostClientCmd(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    
    jr := jresp.NewJsonResp()
    if !auth(r) {
    	jr.Error("Not authorized")
    	fmt.Fprint(w, jr.ToString(debug))
    	return
    }

    // Get client
    registeredClient := server.GetClient(ps.ByName("hostname"))
    if registeredClient == nil {
    	jr.Error("Client not registered")
    	fmt.Fprint(w, jr.ToString(debug))
    	return
    }

    // Timeout
    timeoutStr := r.URL.Query().Get("timeout")
    var timeout int = DEFAULT_COMMAND_TIMEOUT
    if len(strings.TrimSpace(timeoutStr)) > 0 {
    	timeoutI, timeoutE := strconv.ParseInt(timeoutStr, 10, 0)
    	if timeoutE != nil || timeoutI < 1 {
    		jr.Error("Invalid timeout value")
	    	fmt.Fprint(w, jr.ToString(debug))
	    	return
    	}
    	timeout = int(timeoutI)
    }

    // Create command
    command := r.URL.Query().Get("cmd")
    if len(strings.TrimSpace(command)) < 1 {
    	jr.Error("Provide a command")
    	fmt.Fprint(w, jr.ToString(debug))
    	return
    }
    cmd := newCmd(command, timeout) // @todo dynamic

    // Add to list
    registeredClient.mux.Lock()
    registeredClient.Cmds[cmd.Id] = cmd
    registeredClient.CmdChan <- true // Signal for work
    registeredClient.mux.Unlock()

    jr.Set("ack", true)
    jr.OK()
    fmt.Fprint(w, jr.ToString(debug))
}

// Commands
func ClientCmds(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    jr := jresp.NewJsonResp()
    if !auth(r) {
    	jr.Error("Not authorized")
    	fmt.Fprint(w, jr.ToString(debug))
    	return
    }

    // Get client
    registeredClient := server.GetClient(ps.ByName("hostname"))
    if registeredClient == nil {
    	jr.Error("Client not registered")
    	fmt.Fprint(w, jr.ToString(debug))
    	return
    }

    // @todo Read from channel flag and dispatch before timeout
    select {
    case <-registeredClient.CmdChan:
    	cmds := make([]*Cmd, 0)
    	registeredClient.mux.Lock()
        for _, cmd := range registeredClient.Cmds {
        	if cmd.Pending {
        		cmds = append(cmds, cmd)
        		cmd.Pending = false
        	}
        }
        registeredClient.mux.Unlock()
        jr.Set("cmds", cmds)
    case <-time.After(time.Second * LONG_POLL_TIMEOUT):
    	// No commands
        jr.Set("cmds", make([]string, 0))
    }
	jr.OK()
    fmt.Fprint(w, jr.ToString(debug))
}

// Ping
func ClientPing(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    jr := jresp.NewJsonResp()
    if !auth(r) {
    	jr.Error("Not authorized")
    	fmt.Fprint(w, jr.ToString(debug))
    	return
    }
    server.RegisterClient(ps.ByName("hostname"))
	jr.Set("ack", true)
	jr.OK()
    fmt.Fprint(w, jr.ToString(debug))
}

// Ping
func Ping(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !auth(r) {
    	jr.Error("Not authorized")
    	fmt.Fprint(w, jr.ToString(debug))
    	return
    }
	jr.Set("ping", "pong")
	jr.OK()
    fmt.Fprint(w, jr.ToString(debug))
}

// Auth
func auth(r *http.Request) bool {
	if r.Header.Get("X-Auth") != secureToken {
		return false
	}
	return true
}

// Create new server
func newServer() *Server {
	return &Server{
		clients : make(map[string]*RegisteredClient),
	}
}

// New registered client
func newRegisteredClient(hostname string) *RegisteredClient {
	return &RegisteredClient{
		Hostname: hostname,
		Cmds: make(map[string]*Cmd),
		CmdChan: make(chan bool),
	}
}