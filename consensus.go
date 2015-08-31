package main

import (
	"encoding/json"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"sync"
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
	ApproveUserIds map[string]bool
	executeMux     sync.RWMutex
	Executed       bool
}

func (c *Consensus) Get(id string) *ConsensusRequest {
	c.pendingMux.RLock()
	defer c.pendingMux.RUnlock()
	return c.Pending[id]
}

func (c *ConsensusRequest) Cancel(user *User) bool {
	server.consensus.pendingMux.Lock()
	defer server.consensus.pendingMux.Unlock()
	audit.Log(user, "Consensus", fmt.Sprintf("Cancel %s", c.Id))
	delete(server.consensus.Pending, c.Id)
	return true
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

	// Execute
	strategy.Execute(c)

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
	c.pendingMux.Lock()
	defer c.pendingMux.Unlock()
	bytes, je := json.Marshal(c.Pending)
	if je != nil {
		log.Printf("Failed to write consensus: %s", je)
		return
	}
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
			log.Printf("Invalid users.json: %s", je)
			return
		}
		c.Pending = v
	}
}

func (c *Consensus) AddRequest(templateId string, clientIds []string, user *User, reason string) {
	cr := newConsensusRequest()
	cr.TemplateId = templateId
	cr.ClientIds = clientIds
	cr.RequestUserId = user.Id

	audit.Log(user, "Consensus", fmt.Sprintf("Request %s, reason: %s", cr.Id, reason))

	c.pendingMux.Lock()
	c.Pending[cr.Id] = cr
	c.pendingMux.Unlock()
	cr.check()
}

func newConsensus() *Consensus {
	c := &Consensus{
		Pending:  make(map[string]*ConsensusRequest),
		ConfFile: "/etc/indispenso/consensus.json",
	}
	c.load()
	return c
}
func newConsensusRequest() *ConsensusRequest {
	id, _ := uuid.NewV4()
	return &ConsensusRequest{
		Id:             id.String(),
		ApproveUserIds: make(map[string]bool),
	}
}
