package main

import (
	"encoding/json"
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
}

func (c *Consensus) Get(id string) *ConsensusRequest {
	c.pendingMux.RLock()
	defer c.pendingMux.RUnlock()
	return c.Pending[id]
}

func (c *ConsensusRequest) Cancel(user *User) bool {
	server.consensus.pendingMux.Lock()
	defer server.consensus.pendingMux.Unlock()
	delete(server.consensus.Pending, c.Id)
	return true
}

func (c *ConsensusRequest) Approve(user *User) bool {
	if c.RequestUserId == user.Id {
		return false
	}
	if c.ApproveUserIds[user.Id] {
		return false
	}
	c.ApproveUserIds[user.Id] = true

	// @todo Start?

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

func (c *Consensus) AddRequest(templateId string, clientIds []string, requestUserId string) {
	cr := newConsensusRequest()
	cr.TemplateId = templateId
	cr.ClientIds = clientIds
	cr.RequestUserId = requestUserId

	c.pendingMux.Lock()
	c.Pending[cr.Id] = cr
	c.pendingMux.Unlock()
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
		Id: id.String(),
	}
}
