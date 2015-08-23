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
	clientsMux    sync.RWMutex
	clients       map[string]*RegisteredClient
	tagsMux       sync.RWMutex
	Tags          map[string]bool
	userStore     *UserStore
	templateStore *TemplateStore
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

	// Update tags
	s.tagsMux.Lock()
	for _, tag := range tags {
		s.Tags[tag] = true
	}
	s.tagsMux.Unlock()
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
			log.Printf("Client %s disconnected", client.ClientId)
			delete(s.clients, k)
		}
	}
	s.clientsMux.Unlock()
}

type RegisteredClient struct {
	mux      sync.RWMutex
	ClientId string
	LastPing time.Time
	Tags     []string
	Cmds     map[string]*Cmd
	CmdChan  chan bool `json:"-"`
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

	// Templates
	s.templateStore = newTemplateStore()

	// Print info
	log.Printf("Starting server at https://localhost:%d/", serverPort)

	// Start webserver
	go func() {
		router := httprouter.New()
		router.GET("/", Home)
		router.GET("/ping", Ping)
		router.GET("/tags", GetTags)
		router.GET("/client/:clientId/ping", ClientPing)
		router.GET("/client/:clientId/cmds", ClientCmds)
		router.POST("/client/:clientId/cmd", PostClientCmd)
		router.POST("/auth", PostAuth)
		router.GET("/templates", GetTemplate)
		router.POST("/template", PostTemplate)
		router.PUT("/user/password", PutUserPassword)
		router.GET("/clients", GetClients)
		router.GET("/users", GetUsers)
		router.POST("/user", PostUser)
		router.DELETE("/user", DeleteUser)
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

// Get templates
func GetTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	server.templateStore.templateMux.RLock()
	jr.Set("templates", server.templateStore.Templates)
	server.templateStore.templateMux.RUnlock()
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// Create template
func PostTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	user := getUser(r)
	if !user.HasRole("admin") {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	title := strings.TrimSpace(r.PostFormValue("title"))
	description := strings.TrimSpace(r.PostFormValue("description"))
	command := r.PostFormValue("command")
	includedTags := r.PostFormValue("includedTags")
	excludedTags := r.PostFormValue("excludedTags")
	minAuthStr := strings.TrimSpace(r.PostFormValue("minAuth"))
	minAuth, minAuthE := strconv.ParseInt(minAuthStr, 10, 0)
	if len(minAuthStr) < 1 {
		jr.Error("Fill in min auth")
		fmt.Fprint(w, jr.ToString(debug))
		return
	} else if minAuthE != nil {
		jr.Error(fmt.Sprintf("%s", minAuthE))
		fmt.Fprint(w, jr.ToString(debug))
		return
	} else if minAuth < 1 {
		jr.Error("Min auth must be at least 1")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Validate template
	template := newTemplate(title, description, command, true, strings.Split(includedTags, ","), strings.Split(excludedTags, ","), uint(minAuth))
	valid, err := template.IsValid()
	if !valid {
		jr.Error(fmt.Sprintf("%s", err))
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	server.templateStore.Add(template)
	server.templateStore.save()

	jr.Set("saved", true)
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// Login
func PostAuth(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	usr := strings.TrimSpace(r.PostFormValue("username"))
	pwd := strings.TrimSpace(r.PostFormValue("password"))

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
	if !authRes || len(usr) < 1 || len(pwd) < 1 || user == nil || user.Enabled == false {
		jr.Error("Username / password invalid")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	token := user.StartSession()
	user.TouchSession()
	jr.Set("session_token", token)
	roles := make([]string, 0)
	for role := range user.Roles {
		roles = append(roles, role)
	}
	jr.Set("user_roles", roles)
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// List of all tags
func GetTags(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	server.tagsMux.RLock()
	tags := make([]string, 0)
	for tag := range server.Tags {
		tags = append(tags, tag)
	}
	jr.Set("tags", tags)
	server.tagsMux.RUnlock()
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// Change password
func PutUserPassword(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Validate password
	newPwd := r.PostFormValue("password")
	if len(newPwd) < 16 {
		jr.Error("Password must be at least 16 characters, please pick a strong one!")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Match passwords
	newPwd2 := r.PostFormValue("password2")
	if newPwd != newPwd2 {
		jr.Error("Please confirm your password")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Get user
	user := getUser(r)
	if user == nil {
		return
	}

	// Change password
	user.PasswordHash, _ = server.userStore.HashPassword(newPwd)
	server.userStore.save()

	jr.Set("saved", true)
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// User from request
func getUser(r *http.Request) *User {
	// Username
	usr := r.Header.Get("X-Auth-User")

	// Get user
	user := server.userStore.ByName(usr)
	if user == nil {
		return nil
	}

	// Has token?
	if len(user.SessionToken) < 1 {
		return nil
	}

	// Enabled?
	if user.Enabled == false {
		return nil
	}

	// Token expired
	if time.Now().Sub(user.SessionLastTimestamp) > time.Duration(30*time.Minute) {
		return nil
	}

	// Validate token match
	if r.Header.Get("X-Auth-Session") != user.SessionToken {
		return nil
	}
	return user
}

// Delete user
func DeleteUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	usr := getUser(r)
	if !usr.HasRole("admin") {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Get user
	username := r.PostFormValue("username")
	server.userStore.RemoveByName(username)
	server.userStore.save()

	jr.Set("saved", true)
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// Create user
func PostUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	usr := getUser(r)
	if !usr.HasRole("admin") {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Username
	username := r.PostFormValue("username")
	email := r.PostFormValue("email")

	// Validate password
	newPwd := r.PostFormValue("password")
	if len(newPwd) < 16 {
		jr.Error("Password must be at least 16 characters, please pick a strong one!")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Match passwords
	newPwd2 := r.PostFormValue("password2")
	if newPwd != newPwd2 {
		jr.Error("Please confirm your password")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Roles
	roles := strings.Split(r.PostFormValue("roles"), ",")

	// Create user
	res := server.userStore.CreateUser(username, newPwd, email, roles)
	server.userStore.save()

	jr.Set("saved", res)
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// List users
func GetUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	usr := getUser(r)
	if !usr.HasRole("admin") {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}
	server.userStore.usersMux.RLock()
	users := make([]User, 0)
	for _, userPtr := range server.userStore.Users {
		user := *userPtr
		// Hide sensitive fields
		user.PasswordHash = ""
		user.SessionToken = ""
		users = append(users, user)
	}
	jr.Set("users", users)
	server.userStore.usersMux.RUnlock()
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
	clients := make([]*RegisteredClient, 0)
	server.clientsMux.RLock()
	for _, client := range server.clients {
		clients = append(clients, client)
	}
	server.clientsMux.RUnlock()
	jr.Set("clients", clients)
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
	user := getUser(r)
	if user == nil {
		return false
	}
	user.TouchSession()
	return true
}

// Create new server
func newServer() *Server {
	return &Server{
		clients: make(map[string]*RegisteredClient),
		Tags:    make(map[string]bool),
	}
}

// New registered client
func newRegisteredClient(clientId string) *RegisteredClient {
	return &RegisteredClient{
		ClientId: clientId,
		Cmds:     make(map[string]*Cmd),
		CmdChan:  make(chan bool),
	}
}
