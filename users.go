package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/dgryski/dgoogauth"
	"github.com/nu7hatch/gouuid"
	"github.com/oleiade/reflections"
	"golang.org/x/crypto/bcrypt"
	"image/png"
	"io/ioutil"
	"strings"
	"sync"
	"time"
)

const TOTP_MAX_WINDOWS = 3

type AuthType int

const (
	AUTH_TYPE_LOCAL AuthType = 1 << iota
	AUTH_TYPE_LDAP
	AUTH_TYPE_TWO_FACTOR
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
	if len(password) > 0 {
		user.AuthType |= AUTH_TYPE_LOCAL
	}
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
			log.Printf("Invalid user storage file (%s): %s", s.ConfFile, je)
			return
		}
		s.MigrateUsers(v)
		s.Users = v
	}
}

func (s *UserStore) UpdateUser(user *User, changes map[string]interface{}) (err error) {
	u := s.ByName(user.Username)
	u.mux.Lock()
	for k, v := range changes {
		err = reflections.SetField(u, k, v)
	}
	u.mux.Unlock()

	if err != nil {
		return err
	}

	s.save()
	return nil
}

func (s *UserStore) MigrateUsers(users []*User) {
	for _, v := range users {
		if !v.IsAuthDefined() {
			if len(v.PasswordHash) > 0 {
				v.AuthType |= AUTH_TYPE_LOCAL
			}
			if v.LegacyHasTwoFactors() {
				v.AuthType |= AUTH_TYPE_TWO_FACTOR
			}
		}
	}
}

func (s *UserStore) AuthTypes() map[string]int {
	return map[string]int{
		"Local":      int(AUTH_TYPE_LOCAL),
		"LDAP":       int(AUTH_TYPE_LDAP),
		"Two factor": int(AUTH_TYPE_TWO_FACTOR),
	}
}

func (s *UserStore) AddUser(login string, email string, authType AuthType) (*User, error) {
	login = strings.TrimSpace(login)

	// Check unique username
	for _, usr := range s.Users {
		if usr.Username == login {
			return nil, fmt.Errorf("Cannot add new user (%s), user already exsists", login)
		}
	}

	user := newUser()
	user.Username = login
	user.EmailAddress = email
	user.Enabled = true
	user.AuthType |= authType

	s.usersMux.Lock()
	defer s.usersMux.Unlock()
	s.Users = append(s.Users, user)

	return user, nil
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
	AuthType             AuthType
	Enabled              bool
	SessionToken         string
	TotpSecret           string // Secret for time based 2-factor
	TotpSecretValidated  bool   // Did we verify the token?
	SessionIpAddress     string // Current session IP
	SessionLastTimestamp time.Time
	Roles                map[string]bool
	mux                  sync.RWMutex
}

// Two factor setup?
func (u *User) HasTwoFactor() bool {
	u.mux.RLock()
	defer u.mux.RUnlock()
	return u.IsAuthType(AUTH_TYPE_TWO_FACTOR) && u.LegacyHasTwoFactors()
}

func (u *User) LegacyHasTwoFactors() bool {
	return len(u.TotpSecret) > 0 && u.TotpSecretValidated == true
}

func (u *User) IsAuthType(a AuthType) bool {
	return u.AuthType&a != 0
}

func (u *User) IsAuthDefined() bool {
	return u.AuthType != 0
}

// Validate totp token
func (u *User) ValidateTotp(t string) (bool, error) {
	// No token set / provided?
	if len(u.TotpSecret) < 1 || len(strings.TrimSpace(t)) < 1 {
		return false, errors.New("Token not provided")
	}

	// Validate
	cotp := u.otpConfig()
	return cotp.Authenticate(t)
}

func (u *User) otpConfig() dgoogauth.OTPConfig {
	return dgoogauth.OTPConfig{
		Secret:     u.TotpSecret,
		WindowSize: TOTP_MAX_WINDOWS,
	}
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

func (u *User) TouchSession(ip string) {
	u.mux.Lock()
	defer u.mux.Unlock()
	u.SessionLastTimestamp = time.Now()
	u.SessionIpAddress = ip
}

func (u *User) TotpQrImage() ([]byte, error) {
	otpConfig := u.otpConfig()

	// Image uri
	qrCodeImageUri := otpConfig.ProvisionURI(fmt.Sprintf("indispenso:%s", u.Username))

	// QR code
	baseQrImage, qrErr := qr.Encode(qrCodeImageUri, qr.H, qr.Auto)
	if qrErr != nil {
		return nil, fmt.Errorf("Failed to generate QR code: %s", qrErr)
	}

	qrImage, errScale := barcode.Scale(baseQrImage, 300, 300)
	if errScale != nil {
		return nil, fmt.Errorf("Failed to generate QR code scaling problem: %s", errScale)
	}

	var pngQrBuffer bytes.Buffer
	pngQrWriter := bufio.NewWriter(&pngQrBuffer)
	png.Encode(pngQrWriter, qrImage)
	pngQrWriter.Flush()

	return pngQrBuffer.Bytes(), nil
}

func (u *User) GenerateOTPSecret() (err error) {
	// Create TOTP conf
	secret := TotpSecret()
	// Save user, not yet enabled
	u.TotpSecret = secret

	return
}

func (u *User) StartSession() string {
	u.mux.Lock()
	defer u.mux.Unlock()
	u.SessionToken, _ = secureRandomString(32)
	audit.Log(u, "Login", "")
	return u.SessionToken
}

func newUser() *User {
	id, _ := uuid.NewV4()
	return &User{
		Id:    id.String(),
		Roles: make(map[string]bool),
	}
}

func newUserStore(confFile string) *UserStore {
	store := &UserStore{
		Users:    make([]*User, 0),
		ConfFile: confFile,
	}
	store.load()
	store.prepareDefaultUser()
	return store
}
