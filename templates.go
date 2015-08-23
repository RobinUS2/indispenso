package main

import (
	"encoding/json"
	"io/ioutil"
	"sync"
)

// Templates used to be executed on hosts

type Template struct {
	Id          string
	Title       string // Short title
	Description string // Full description that explains in layman's terms what this does, so everyone can help as part of the authorization process
	Enabled     bool   // Is this available for running?
}

type TemplateStore struct {
	ConfFile    string
	Templates   map[string]*Template
	templateMux sync.RWMutex
}

func (s *TemplateStore) save() {
	s.templateMux.Lock()
	defer s.templateMux.Unlock()
	bytes, je := json.Marshal(s.Templates)
	if je != nil {
		log.Printf("Failed to write templates: %s", je)
		return
	}
	err := ioutil.WriteFile(s.ConfFile, bytes, 0644)
	if err != nil {
		log.Printf("Failed to write templates: %s", err)
		return
	}
}

func (s *TemplateStore) load() {
	s.templateMux.Lock()
	defer s.templateMux.Unlock()
	// Read file and load into user store
	bytes, err := ioutil.ReadFile(s.ConfFile)
	if err == nil {
		var v map[string]*Template
		je := json.Unmarshal(bytes, &v)
		if je != nil {
			log.Printf("Invalid templates.json: %s", je)
			return
		}
		s.Templates = v
	}
}

func newTemplateStore() *TemplateStore {
	s := &TemplateStore{
		ConfFile:  "/etc/indispenso/templates.conf",
		Templates: make(map[string]*Template),
	}
	s.load()
	return s
}
