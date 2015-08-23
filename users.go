package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/nu7hatch/gouuid"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"strings"
	"sync"
	"time"
)

// Users

type UserStore struct {
	usersMux sync.RWMutex
	Users    []*User
	ConfFile string
}

func (s *UserStore) ByName(username string) *User {
	s.usersMux.Lock()
	defer s.usersMux.Unlock()

	for _, user := range s.Users {
		if user.Username == username {
			return user
		}
	}
	return nil
}

func (s *UserStore) RemoveByName(username string) {
	s.usersMux.Lock()
	defer s.usersMux.Unlock()
	tmp := make([]*User, 0)
	for _, user := range s.Users {
		if user.Username == username {

			continue
		}
		tmp = append(tmp, user)
	}
	s.Users = tmp
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

func (s *UserStore) Auth(hash string, pwd string) bool {
	bytes, be := base64.URLEncoding.DecodeString(hash)
	if be != nil {
		log.Printf("%s", be)
		bytes = make([]byte, 0)
	}
	return bcrypt.CompareHashAndPassword(bytes, []byte(pwd)) == nil
}

func (s *UserStore) CreateUser(username string, password string, email string, roles []string) bool {
	s.usersMux.Lock()
	defer s.usersMux.Unlock()

	user := newUser()
	user.Username = strings.TrimSpace(username)
	user.EmailAddress = email
	user.Enabled = true

	// Roles
	for _, role := range roles {
		user.Roles[role] = true
	}

	// Check unique username
	for _, usr := range s.Users {
		if usr.Username == user.Username {
			return false
		}
	}

	hash, e := s.HashPassword(password)
	if e != nil {
		log.Fatal("Failed to hash password")
		return false
	}
	user.PasswordHash = hash
	s.Users = append(s.Users, user)
	return true
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
		pwd, _ := secureRandomString(32)
		log.Println("You don't have an admin user yet, creating with the following password:")
		log.Println(pwd)

		s.CreateUser("admin", pwd, "", make([]string, 0))

		// Elevate to admin rights
		usr := s.ByName("admin")
		usr.AddRole("admin")
		usr.AddRole("requester")
		usr.AddRole("approver")

		// Save and reload
		s.save()
		s.load()
	}
}

type User struct {
	Id                   string
	Username             string
	EmailAddress         string
	PasswordHash         string
	Enabled              bool
	SessionToken         string
	SessionLastTimestamp time.Time
	Roles                map[string]bool
	mux                  sync.RWMutex
}

func (u *User) HasRole(r string) bool {
	u.mux.RLock()
	defer u.mux.RUnlock()
	return u.Roles[r]
}

func (u *User) AddRole(r string) {
	u.mux.Lock()
	defer u.mux.Unlock()
	u.Roles[r] = true
}

func (u *User) TouchSession() {
	u.mux.Lock()
	defer u.mux.Unlock()
	u.SessionLastTimestamp = time.Now()
}

func (u *User) StartSession() string {
	u.mux.Lock()
	defer u.mux.Unlock()
	u.SessionToken, _ = secureRandomString(32)
	return u.SessionToken
}

func newUser() *User {
	id, _ := uuid.NewV4()
	return &User{
		Id:    id.String(),
		Roles: make(map[string]bool),
	}
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
