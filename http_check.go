package main

// Using a template as an http check is possible where you can monitor the end point externally to validate your systems are running perfectly
// @author Robin Verlangen

import (
	"encoding/json"
	"fmt"
	"github.com/RobinUS2/golang-jresp"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Http checks
type HttpCheckStore struct {
	Checks     map[string]*HttpCheckConfiguration
	ConfFile   string
	SystemUser *User
	mux        sync.RWMutex
}

// An HTTP check consist of a template and a set of hosts to run on
type HttpCheckConfiguration struct {
	Id          string
	Enabled     bool
	TemplateId  string
	SecureToken string
	Timeout     int
	ClientIds   []string
}

// Http handler for the server
func GetHttpCheck(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// No auth for the user / server, this is accessible externally
	jr := jresp.NewJsonResp()
	id := ps.ByName("id")

	// Get check and make sure it is active
	c := server.httpCheckStore.Get(id)
	if c == nil || c.Enabled == false {
		jr.Error("Check not found")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Validate token
	token := r.URL.Query().Get("token")
	if len(token) < 1 || token != c.SecureToken {
		jr.Error("Secure token invalid")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Execute the config
	cr := server.consensus.AddRequest(c.TemplateId, c.ClientIds, server.httpCheckStore.SystemUser, "")
	if cr == nil {
		jr.Error("Unable to start check")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Register callback
	done := make(chan bool, 1)
	cb := func(cr *ConsensusRequest) {
		done <- true
	}
	cr.AddCallback(cb)

	// Trigger execution
	cr.check()

	// Wait for success (or failure..)
	select {
	case <-time.After(time.Duration(c.Timeout) * time.Second):
		jr.Error("Timeout")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	case <-done:
	}

	// Cleanup
	cr.Delete()

	// Print results
	jr.OK()
	fmt.Fprint(w, jr.ToString(conf.Debug))
}

// Get item
func (s *HttpCheckStore) Get(id string) *HttpCheckConfiguration {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.Checks[id]
}

// Find by template id
func (s *HttpCheckStore) FindByTemplate(id string) []*HttpCheckConfiguration {
	list := make([]*HttpCheckConfiguration, 0)
	s.mux.RLock()
	defer s.mux.RUnlock()
	for _, elm := range s.Checks {
		// Does it match?
		if elm.TemplateId != id {
			continue
		}
		list = append(list, elm)
	}
	return list
}

// Add item
func (s *HttpCheckStore) Add(e *HttpCheckConfiguration) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.Checks[e.Id] = e
}

// Remove item
func (s *HttpCheckStore) Remove(id string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	tmp := make(map[string]*HttpCheckConfiguration)
	for k, elm := range s.Checks {
		if elm.Id == id {
			continue
		}
		tmp[k] = elm
	}
	s.Checks = tmp
}

// Save to disk
func (s *HttpCheckStore) save() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	bytes, je := json.Marshal(s.Checks)
	if je != nil {
		log.Printf("Failed to write http checks: %s", je)
		return false
	}
	err := ioutil.WriteFile(s.ConfFile, bytes, 0644)
	if err != nil {
		log.Printf("Failed to write http checks: %s", err)
		return false
	}
	return true
}

// List HTTP Checks
func GetHttpChecks(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Must be admin
	user := getUser(r)
	if !user.HasRole("admin") {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}
	server.httpCheckStore.mux.RLock()
	jr.Set("checks", server.httpCheckStore.Checks)
	server.httpCheckStore.mux.RUnlock()
	jr.OK()
	fmt.Fprint(w, jr.ToString(conf.Debug))
}

// Delete HTTP check
func DeleteHttpCheck(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Must be admin
	user := getUser(r)
	if !user.HasRole("admin") {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Remove
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	audit.Log(user, "HTTP check", fmt.Sprintf("Deleted %s", id))
	server.httpCheckStore.Remove(id)

	// Save
	res := server.httpCheckStore.save()
	jr.Set("saved", res)
	jr.OK()
	fmt.Fprint(w, jr.ToString(conf.Debug))
}

// Create HTTP Check
func PostHttpCheck(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	if !authUser(r) {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Must be admin
	user := getUser(r)
	if !user.HasRole("admin") {
		jr.Error("Not authorized")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Verify two factor for, so that a hacked account can not request or execute anything without getting access to the 2fa device
	if res, _ := user.ValidateTotp(r.PostFormValue("totp")); res == false {
		jr.Error("Invalid two factor token")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Template
	templateId := strings.TrimSpace(r.PostFormValue("template"))
	template := server.templateStore.Get(templateId)
	if template == nil {
		jr.Error("Template not found")
		fmt.Fprint(w, jr.ToString(conf.Debug))
		return
	}

	// Client IDs
	clientIds := strings.Split(strings.TrimSpace(r.PostFormValue("clients")), ",")

	// Create
	hc := newHttpCheckConfiguration()
	hc.ClientIds = clientIds
	hc.TemplateId = templateId
	hc.Enabled = true
	hc.Timeout = 30

	// Add and save
	server.httpCheckStore.Add(hc)
	server.httpCheckStore.save()

	// Done
	jr.OK()
	fmt.Fprint(w, jr.ToString(conf.Debug))
}

// Load from disk
func (s *HttpCheckStore) load() {
	s.mux.Lock()
	defer s.mux.Unlock()
	// Read file and load into user store
	bytes, err := ioutil.ReadFile(s.ConfFile)
	if err == nil {
		var v map[string]*HttpCheckConfiguration
		je := json.Unmarshal(bytes, &v)
		if je != nil {
			log.Printf("Invalid httpchecks.json: %s", je)
			return
		}
		s.Checks = v
	}
}

// New store
func newHttpCheckStore() *HttpCheckStore {
	systemUser := newUser()
	systemUser.AddRole("requester")
	s := &HttpCheckStore{
		ConfFile:   conf.HomeFile("httpchecks.json"),
		Checks:     make(map[string]*HttpCheckConfiguration),
		SystemUser: systemUser,
	}
	s.load()
	return s
}

// New check
func newHttpCheckConfiguration() *HttpCheckConfiguration {
	token, _ := secureRandomString(32)
	return &HttpCheckConfiguration{
		Id:          uuidStr(),
		Timeout:     30,
		Enabled:     true,
		SecureToken: token,
	}
}
