package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/RobinUS2/golang-jresp"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Server methods (you probably only need one or two in HA failover mode)

type Server struct {
	clientsMux sync.RWMutex
	clients    map[string]*RegisteredClient
	userStore  *UserStore
}

// Register client
func (s *Server) RegisterClient(clientId string, tags []string) {
	s.clientsMux.Lock()
	if s.clients[clientId] == nil {
		s.clients[clientId] = newRegisteredClient(clientId)
		log.Printf("Client %s registered with tags %s", clientId, tags)
	}
	s.clients[clientId].mux.Lock()
	s.clients[clientId].LastPing = time.Now()
	s.clients[clientId].Tags = tags
	s.clients[clientId].mux.Unlock()
	s.clientsMux.Unlock()
}

func (s *Server) GetClient(clientId string) *RegisteredClient {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	return s.clients[clientId]
}

// Scan for old clients
func (s *Server) CleanupClients() {
	s.clientsMux.Lock()
	for k, client := range s.clients {
		if time.Now().Sub(client.LastPing).Seconds() > float64(CLIENT_PING_INTERVAL*5) {
			// Disconnect
			log.Printf("Client %s disconnected", client.clientId)
			delete(s.clients, k)
		}
	}
	s.clientsMux.Unlock()
}

type RegisteredClient struct {
	mux      sync.RWMutex
	clientId string
	LastPing time.Time
	Tags     []string
	Cmds     map[string]*Cmd
	CmdChan  chan bool
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
	// Users
	s.userStore = newUserStore()

	// Print info
	log.Printf("Starting server at https://localhost:%d/", serverPort)

	// Start webserver
	go func() {
		router := httprouter.New()
		router.GET("/", Home)
		router.GET("/ping", Ping)
		router.GET("/client/:clientId/ping", ClientPing)
		router.GET("/client/:clientId/cmds", ClientCmds)
		router.POST("/client/:clientId/cmd", PostClientCmd)
		router.POST("/auth", PostAuth)
		router.GET("/clients", GetClients)
		router.ServeFiles("/console/*filepath", http.Dir("console"))

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

// Login
func PostAuth(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	usr := r.PostFormValue("username")
	pwd := r.PostFormValue("password")

	// Fetch user
	user := server.userStore.ByName(usr)

	// Hash and check (also if there is no user to prevent timing attacks)
	hash := ""
	if user != nil {
		hash = user.PasswordHash
	} else {
		// Fake password
		hash = "JDJhJDExJDBnOVJ4cmo4OHhzeGliV2oucDFrLmUzQlYzN296OVBlU1JqNU1FVWNqVGVCZEEuaWtMS2oo"
	}
	authRes := server.userStore.Auth(hash, pwd)
	if !authRes {
		jr.Error("Username / password invalid")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	token := user.StartSession()
	user.TouchSession()
	jr.Set("session_token", token)
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// List clients
func GetClients(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
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
	registeredClient := server.GetClient(ps.ByName("clientId"))
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
	registeredClient := server.GetClient(ps.ByName("clientId"))
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
	tags := strings.Split(r.URL.Query().Get("tags"), ",")
	server.RegisterClient(ps.ByName("clientId"), tags)
	jr.Set("ack", true)
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// Home
func Home(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Redirect to console
	http.Redirect(w, r, r.URL.String()+"console/", 301)
}

// Ping
func Ping(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	jr := jresp.NewJsonResp()
	jr.Set("ping", "pong")
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// Auth
func auth(r *http.Request) bool {
	// Signed token
	uri := r.URL.String()
	hasher := sha256.New()
	hasher.Write([]byte(uri))
	signedToken := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	// Validate
	if r.Header.Get("X-Auth") != signedToken {
		return false
	}
	return true
}

// Auth user
func authUser(r *http.Request) bool {
	// Username
	usr := r.Header.Get("X-Auth-User")

	// Get user
	user := server.userStore.ByName(usr)
	if user == nil {
		return false
	}

	// Has token?
	if len(user.SessionToken) < 1 {
		return false
	}

	// Token expired
	if time.Now().Sub(user.SessionLastTimestamp) > time.Duration(30*time.Minute) {
		return false
	}

	// Validate token match
	if r.Header.Get("X-Auth-Session") != user.SessionToken {
		return false
	}
	return true
}

// Create new server
func newServer() *Server {
	return &Server{
		clients: make(map[string]*RegisteredClient),
	}
}

// New registered client
func newRegisteredClient(clientId string) *RegisteredClient {
	return &RegisteredClient{
		clientId: clientId,
		Cmds:     make(map[string]*Cmd),
		CmdChan:  make(chan bool),
	}
}
