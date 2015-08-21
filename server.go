package main

import (
	"net/http"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/RobinUS2/golang-jresp"
	"sync"
	"time"
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

	    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverPort), router))
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

// Commands
func ClientCmds(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    jr := jresp.NewJsonResp()
    if !auth(r) {
    	jr.Error("Not authorized")
    	fmt.Fprint(w, jr.ToString(debug))
    	return
    }
    time.Sleep(1 * time.Second)
    server.RegisterClient(ps.ByName("hostname"))
	jr.Set("cmds", make([]string, 0))
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
	}
}