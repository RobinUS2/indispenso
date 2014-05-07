// @author Robin Verlangen
// User handler

package main

// Imports
import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"github.com/dgryski/dgoogauth"
	"io"
	"log"
	"math/rand"
	"strings"
)

// Defaults
const DEFAULT_ADMIN_USR = "admin"
const DEFUALT_ADMIN_PWD = "indispenso"
const MIN_USR_LEN int = 3
const MIN_PWD_LEN int = 8

// User handler
type UserHandler struct {
}

// User
type User struct {
	Id            string // Uuid string
	Username      string
	PasswordHash  string // Sha 512 hash concat(pwd, salt)
	PasswordSalt  string
	IsAdmin       bool // Permission: system management
	IsRequester   bool // Permission: request task execution
	IsApprover    bool // Permission: approve task execution
	TwoFactorSeed string
}

// Display name
func (u *User) DisplayName() string {
	return u.Username
}

// Check two-factor
func (u *User) IsValidTwoFactor(token string) bool {
	// Do we have two factor enabled?
	if len(u.TwoFactorSeed) == 0 {
		// No, OK pass
		log.Println(fmt.Sprintf("WARN: User %s login without two-factor enabled", u.DisplayName()))
		return true
	}

	// Configure token
	var cotp *dgoogauth.OTPConfig = &dgoogauth.OTPConfig{
		Secret:     u.TwoFactorSeed,
		WindowSize: 3,
	}

	// Validate token
	authRes, authErr := cotp.Authenticate(token)
	if authErr != nil || authRes == false {
		log.Println(fmt.Sprintf("WARN: User %s failed two-factor challenge, invalid token", u.DisplayName()))
		return false
	} else {
		// OK
		return true
	}

	// By default error
	return false
}

// New user
func NewUser(username string, password string, isAdmin bool, isRequester bool, isApprover bool) *User {
	// Hash password with random salt
	var salt string = HashPassword(fmt.Sprintf("%d", rand.Int63()), "")
	hash := HashPassword(password, salt)

	// Struct
	return &User{
		Id:           getUuid(),
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
		IsAdmin:      isAdmin,
		IsRequester:  isRequester,
		IsApprover:   isApprover,
	}
}

// Hash password
func HashPassword(pwd string, salt string) string {
	h := sha512.New()
	io.WriteString(h, pwd)
	io.WriteString(h, salt)
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return hash
}

// New user handler
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// Create user
func (u *UserHandler) CreateUser(username string, password string, isAdmin bool, isRequester bool, isApprover bool) (*User, error) {
	// Basic validation
	username = strings.TrimSpace(username)
	if len(username) < MIN_USR_LEN {
		return nil, newErr(fmt.Sprintf("Please provide a username of at least %d characters", MIN_USR_LEN))
	}
	password = strings.TrimSpace(password)
	if len(password) < MIN_PWD_LEN {
		return nil, newErr(fmt.Sprintf("Please provide a password of at least %d characters", MIN_PWD_LEN))
	}

	// Existing user, do NOT use GetUser, this causes recursive StackOverflow
	existing := u.GetUserData(username)
	if existing != nil {
		return nil, newErr(fmt.Sprintf("Username '%s' already taken", username))
	}

	// Create struct
	user := NewUser(username, password, isAdmin, isRequester, isApprover)

	// Save in cluster (not async)
	k := fmt.Sprintf("user~%s", user.Username)
	b, err := json.Marshal(user)
	if err != nil {
		return nil, newErr(fmt.Sprintf("ERR: Failed to convert user struct to json %s", err))
	}
	if datastore == nil {
		return nil, newErr(fmt.Sprintf("ERR: Datastore not available"))
	}
	if datastore.PutEntry(k, string(b)) {
		log.Println(fmt.Sprintf("INFO: Created user %s", user.Username))
	}

	// Done
	return user, nil
}

// Get user
func (u *UserHandler) GetUser(username string) *User {
	// Fetch
	entry := u.GetUserData(username)
	if entry == nil {
		if username == DEFAULT_ADMIN_USR {
			log.Println("INFO: Creating default admin user")
			newAdmin, newAdminErr := u.CreateUser(DEFAULT_ADMIN_USR, DEFUALT_ADMIN_PWD, true, true, true)
			if newAdminErr == nil {
				return newAdmin
			}
		}
		return nil
	}

	// Convert to struct
	var user *User
	err := json.Unmarshal([]byte(entry.Value), &user)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to unmarshal user %s", err))
		return nil
	}

	// Return
	return user
}

// Get user data
func (u *UserHandler) GetUserData(username string) *MemEntry {
	if datastore == nil {
		log.Println(fmt.Sprintf("ERR: Datastore not available"))
		return nil
	}
	k := fmt.Sprintf("user~%s", username)
	e, _ := datastore.GetEntry(k)
	return e
}
