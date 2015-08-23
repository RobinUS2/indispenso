package main

import (
	"github.com/nu7hatch/gouuid"
	"sync"
)

// @author Robin Verlangen
// This element will controll the requests and let people vote untill the authorization level is met

type Consensus struct {
	pendingMux sync.RWMutex
	Pending    map[string]*ConsensusRequest
}

type ConsensusRequest struct {
	Id            string
	TemplateId    string
	ClientIds     []string
	RequestUserId string
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
	return &Consensus{
		Pending: make(map[string]*ConsensusRequest),
	}
}
func newConsensusRequest() *ConsensusRequest {
	id, _ := uuid.NewV4()
	return &ConsensusRequest{
		Id: id.String(),
	}
}
