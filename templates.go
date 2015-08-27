package main

import (
	"encoding/json"
	"errors"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"sync"
)

// Templates used to be executed on hosts

type Template struct {
	Id              string
	Title           string // Short title
	Description     string // Full description that explains in layman's terms what this does, so everyone can help as part of the authorization process
	Command         string // Command to be executed
	Enabled         bool   // Is this available for running?
	Timeout         int    // Seconds of execution before the command is killed
	Acl             *TemplateACL
	ValidationRules []*ExecutionValidation // Validation rules
	mux             sync.RWMutex
}

type TemplateACL struct {
	MinAuth      uint // Minimum amount of authorization before the template is actually executed (eg 3 = requester + 2 additional approvers)
	IncludedTags []string
	ExcludedTags []string
}

type TemplateStore struct {
	ConfFile    string
	Templates   map[string]*Template
	templateMux sync.RWMutex
}

// Add a validation rule
func (s *Template) AddValidationRule(r *ExecutionValidation) {
	s.ValidationRules = append(s.ValidationRules, r)
}

// Delete a validation rule
func (s *Template) DeleteValidationRule(id string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	var tmp = make([]*ExecutionValidation, 0)
	for _, r := range s.ValidationRules {
		if r.Id == id {
			continue
		}
		tmp = append(tmp, r)
	}
	s.ValidationRules = tmp
}

// Validate the setup of a template
func (s *Template) IsValid() (bool, error) {
	if len(s.Title) < 1 {
		return false, errors.New("Fill in a title")
	}

	// Title must be unique
	server.templateStore.templateMux.RLock()
	defer server.templateStore.templateMux.RUnlock()
	for _, template := range server.templateStore.Templates {
		if template.Title == s.Title {
			return false, errors.New("Title is not unique")
		}
	}

	if len(s.Description) < 1 {
		return false, errors.New("Fill in a description")
	}
	if len(s.Command) < 1 {
		return false, errors.New("Fill in a command")
	}
	return true, nil
}

func (s *TemplateStore) Remove(templateId string) {
	s.templateMux.Lock()
	defer s.templateMux.Unlock()
	delete(s.Templates, templateId)
}

func (s *TemplateStore) Get(templateId string) *Template {
	s.templateMux.RLock()
	defer s.templateMux.RUnlock()
	return s.Templates[templateId]
}

func (s *TemplateStore) Add(template *Template) {
	s.templateMux.Lock()
	defer s.templateMux.Unlock()
	s.Templates[template.Id] = template
}

func (s *TemplateStore) save() bool {
	s.templateMux.Lock()
	defer s.templateMux.Unlock()
	bytes, je := json.Marshal(s.Templates)
	if je != nil {
		log.Printf("Failed to write templates: %s", je)
		return false
	}
	err := ioutil.WriteFile(s.ConfFile, bytes, 0644)
	if err != nil {
		log.Printf("Failed to write templates: %s", err)
		return false
	}
	return true
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

func newTemplateAcl() *TemplateACL {
	return &TemplateACL{
		IncludedTags: make([]string, 0),
		ExcludedTags: make([]string, 0),
	}
}

func newTemplate(title string, description string, command string, enabled bool, includedTags []string, excludedTags []string, minAuth uint, timeout int) *Template {
	// Unique ID
	id, _ := uuid.NewV4()

	// ACL
	acl := newTemplateAcl()
	acl.IncludedTags = includedTags
	acl.ExcludedTags = excludedTags
	acl.MinAuth = minAuth

	// Tags
	if len(acl.IncludedTags) == 1 && acl.IncludedTags[0] == "" {
		acl.IncludedTags = make([]string, 0)
	}
	if len(acl.ExcludedTags) == 1 && acl.ExcludedTags[0] == "" {
		acl.ExcludedTags = make([]string, 0)
	}

	// Instantiate
	t := &Template{
		Id:              id.String(),
		Title:           title,
		Description:     description,
		Command:         command,
		Enabled:         enabled,
		Acl:             acl,
		Timeout:         timeout,
		ValidationRules: make([]*ExecutionValidation, 0),
	}

	return t
}
