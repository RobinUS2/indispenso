package main

import (
	"encoding/json"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"sync"
	"time"
)

// @author Robin Verlangen
// This element will controll the requests and let people vote untill the authorization level is met

type Consensus struct {
	pendingMux sync.RWMutex
	Pending    map[string]*ConsensusRequest
	ConfFile   string
}

type ConsensusRequest struct {
	Id             string
	TemplateId     string
	ClientIds      []string
	RequestUserId  string
	Reason         string
	ApproveUserIds map[string]bool
	executeMux     sync.RWMutex
	Executed       bool
	CreateTime     int64                     // Unix TS for creation of consensus request
	StartTime      int64                     // Unix TS for start of command execution
	CompleteTime   int64                     // Unix TS for completion of command exectuion
	Callbacks      []func(*ConsensusRequest) `json:"-"` // Will be called on completions
}

func (c *Consensus) Get(id string) *ConsensusRequest {
	c.pendingMux.RLock()
	defer c.pendingMux.RUnlock()
	return c.Pending[id]
}

// Delete
func (c *ConsensusRequest) Delete() bool {
	// Delete child commands
	server.clientsMux.RLock()
	for _, client := range server.clients {
		client.mux.Lock()
		for k, cmd := range client.DispatchedCmds {
			if cmd.ConsensusRequestId == c.Id {
				delete(client.DispatchedCmds, k)
			}
		}
		client.mux.Unlock()
	}
	server.clientsMux.RUnlock()

	// Delete request itself
	server.consensus.pendingMux.Lock()
	defer server.consensus.pendingMux.Unlock()
	delete(server.consensus.Pending, c.Id)

	return true
}

// Cancel the request
func (c *ConsensusRequest) Cancel(user *User) bool {
	audit.Log(user, "Consensus", fmt.Sprintf("Cancel %s", c.Id))
	return c.Delete()
}
func (c *ConsensusRequest) Template() *Template {
	server.templateStore.templateMux.RLock()
	template := server.templateStore.Templates[c.TemplateId]
	server.templateStore.templateMux.RUnlock()
	return template
}

// Start template execution
func (c *ConsensusRequest) start() bool {
	template := c.Template()
	if template == nil {
		log.Printf("Template %s not found for request %s", c.TemplateId, c.Id)
		return false
	}

	// Lock
	c.executeMux.Lock()
	defer c.executeMux.Unlock()
	if c.Executed {
		// Already executed
		return false
	}
	c.Executed = true

	// Currently we only support one execution strategy
	strategy := c.Template().GetExecutionStrategy()
	if strategy == nil {
		log.Printf("Execution strategy not found for request %s", c.Id)
		return false
	}

	// Start time
	c.StartTime = time.Now().Unix()

	// Execute
	strategy.Execute(c)

	// Completed
	c.CompleteTime = time.Now().Unix()

	return true
}

// Check whether this request is good to dispatch
func (c *ConsensusRequest) check() bool {
	// Can we start?
	template := c.Template()
	if template == nil {
		log.Printf("Template %s not found for request %s", c.TemplateId, c.Id)
		return false
	}

	// Did we meet the auth?
	minAuth := template.Acl.MinAuth
	voteCount := 1 // Initial vote by the requester
	for _ = range c.ApproveUserIds {
		voteCount++
	}
	if uint(voteCount) < minAuth {
		// Did not meet
		log.Printf("Vote count %d does not yet meet required %d for request %s", voteCount, minAuth, c.Id)
		return false
	}

	// Start
	return c.start()
}

func (c *ConsensusRequest) Approve(user *User) bool {
	if c.ApproveUserIds == nil {
		c.ApproveUserIds = make(map[string]bool)
	}
	if c.RequestUserId == user.Id {
		return false
	}
	if c.ApproveUserIds[user.Id] {
		return false
	}
	c.ApproveUserIds[user.Id] = true

	audit.Log(user, "Consensus", fmt.Sprintf("Approve %s", c.Id))

	c.check()

	return true
}

func (c *Consensus) save() {
	// Lock
	c.pendingMux.Lock()
	defer c.pendingMux.Unlock()

	// Cleanup older than 2 weeks
	maxAge := time.Now().Unix() - (14 * 86400)
	newPending := make(map[string]*ConsensusRequest)
	for k, pending := range c.Pending {
		// Skip if too old
		if pending.CreateTime < maxAge {
			continue
		}
		newPending[k] = pending
	}

	// Put in place
	c.Pending = newPending

	// To JSON
	bytes, je := json.Marshal(c.Pending)
	if je != nil {
		log.Printf("Failed to write consensus: %s", je)
		return
	}

	// Write to disk
	err := ioutil.WriteFile(c.ConfFile, bytes, 0644)
	if err != nil {
		log.Printf("Failed to write consensus: %s", err)
		return
	}
}

func (c *Consensus) load() {
	c.pendingMux.Lock()
	defer c.pendingMux.Unlock()
	// Read file and load into
	bytes, err := ioutil.ReadFile(c.ConfFile)
	if err == nil {
		var v map[string]*ConsensusRequest
		je := json.Unmarshal(bytes, &v)
		if je != nil {
			log.Printf("Invalid consensus storage file (%s) due to: %s", c.ConfFile, je)
			return
		}
		c.Pending = v
	}
}

func (c *Consensus) AddRequest(templateId string, clientIds []string, user *User, reason string) *ConsensusRequest {
	// Double check permissions
	if !user.HasRole("requester") {
		log.Printf("User %s (%s) does not have requester permissions", user.Username, user.Id)
		return nil
	}

	// Create request
	cr := newConsensusRequest()
	cr.TemplateId = templateId
	cr.ClientIds = clientIds
	cr.RequestUserId = user.Id
	cr.Reason = reason

	message := fmt.Sprintf("Request %s, reason: %s", cr.Id, cr.Reason)
	audit.Log(user, "Consensus", message)

	c.pendingMux.Lock()
	c.Pending[cr.Id] = cr
	c.pendingMux.Unlock()

	server.notifications.Notify(&Message{Type: NEW_CONSENSUS, Content: message, Url: conf.ServerRequest("/console/#!pending")})

	return cr
}

func newConsensus() *Consensus {
	c := &Consensus{
		Pending:  make(map[string]*ConsensusRequest),
		ConfFile: conf.HomeFile("consensus.json"),
	}
	c.load()
	return c
}

func consensusRequestFinishedNotification(consensusRequest *ConsensusRequest) {
	server.notifications.Notify(&Message{Type: EXECUTION_DONE, Content: "", Url: conf.ServerRequest("/console/#!pending")})
}

func newConsensusRequest() *ConsensusRequest {
	id, _ := uuid.NewV4()
	return &ConsensusRequest{
		Id:             id.String(),
		ApproveUserIds: make(map[string]bool),
		CreateTime:     time.Now().Unix(),
		Callbacks:      []func(*ConsensusRequest){consensusRequestFinishedNotification},
	}
}
