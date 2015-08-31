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
	Id         string
	Enabled    bool
	TemplateId string
	ClientIds  []string
}

// Http handler for the server
func GetHttpCheck(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jr := jresp.NewJsonResp()
	id := ps.ByName("id")

	// Get check and make sure it is active
	c := server.httpCheckStore.Get(id)
	if c == nil || c.Enabled == false {
		jr.Error("Check not found")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Execute the config
	cr := server.consensus.AddRequest(c.TemplateId, c.ClientIds, server.httpCheckStore.SystemUser, "")
	if cr == nil {
		jr.Error("Unable to start check")
		fmt.Fprint(w, jr.ToString(debug))
		return
	}

	// Register callback
	done := make(chan bool, 1)
	cb := func(cr *ConsensusRequest) {
		done <- true
	}
	cr.Callbacks = append(cr.Callbacks, cb)

	// Trigger execution
	cr.check()

	// Wait for success (or failure..)
	select {
	case <-time.After(30 * time.Second):
		jr.Error("Timeout")
		fmt.Fprint(w, jr.ToString(debug))
		return
	case <-done:
	}

	// Print results
	jr.OK()
	fmt.Fprint(w, jr.ToString(debug))
}

// Get item
func (s *HttpCheckStore) Get(id string) *HttpCheckConfiguration {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.Checks[id]
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

func newHttpCheckStore() *HttpCheckStore {
	systemUser := newUser()
	systemUser.AddRole("requester")
	s := &HttpCheckStore{
		ConfFile:   "/etc/indispenso/httpchecks.json",
		Checks:     make(map[string]*HttpCheckConfiguration),
		SystemUser: systemUser,
	}
	s.load()
	return s
}
