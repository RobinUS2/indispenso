package main

import (
	"encoding/base64"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"sync"
)

// Users

type UserStore struct {
	usersMux sync.RWMutex
	Users    []*User
	ConfFile string
}

func (s *UserStore) save() {
	s.usersMux.Lock()
	defer s.usersMux.Unlock()
	bytes, je := json.Marshal(s.Users)
	if je != nil {
		log.Printf("Failed to write users: %s", je)
		return
	}
	err := ioutil.WriteFile(s.ConfFile, bytes, 0644)
	if err != nil {
		log.Printf("Failed to write users: %s", err)
		return
	}
}

func (s *UserStore) CreateUser(username string, password string) {
	s.usersMux.Lock()
	defer s.usersMux.Unlock()
	user := newUser()
	user.Username = username
	hash, e := s.HashPassword(password)
	if e != nil {
		log.Fatal("Failed to hash password")
		return
	}
	user.PasswordHash = hash
	s.Users = append(s.Users, user)
}

func (s *UserStore) HashPassword(pwd string) (string, error) {
	b, e := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost+1)
	str := base64.URLEncoding.EncodeToString(b)
	return str, e
}

func (s *UserStore) load() {
	s.usersMux.Lock()
	defer s.usersMux.Unlock()
	// Read file and load into user store
	bytes, err := ioutil.ReadFile(s.ConfFile)
	if err == nil {
		var v []*User
		je := json.Unmarshal(bytes, &v)
		if je != nil {
			log.Printf("Invalid users.json: %s", je)
			return
		}
		s.Users = v
	}
}

func (s *UserStore) prepareDefaultUser() {
	if s.Users == nil || len(s.Users) < 1 {
		s.CreateUser("admin", "indispenso")
		s.save()
		s.load()
	}
}

type User struct {
	Username     string
	PasswordHash string
	Enabled      bool
}

func newUser() *User {
	return &User{}
}

func newUserStore() *UserStore {
	store := &UserStore{
		Users:    make([]*User, 0),
		ConfFile: "/etc/indispenso/users.json",
	}
	store.load()
	store.prepareDefaultUser()
	return store
}
