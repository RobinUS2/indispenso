// @author Robin Verlangen
// User handler

package main

// Imports
import (
	"fmt"
	"log"
	"encoding/json"
	"strings"
	"io"
	"crypto/sha512"
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
	Id string // Uuid string
	Username string
	PasswordHash string // Sha 512 hash concat(pwd, salt)
	PasswordSalt string
	IsAdmin bool // Permission: system management
	IsRequester bool // Permission: request task execution
	IsApprover bool// Permission: approve task execution
}

// New user
func NewUser(username string, password string, isAdmin bool, isRequester bool, isApprover bool) *User {
	// Hash password
	var salt string = "1234ab"
	h := sha512.New()
	io.WriteString(h, password)
	io.WriteString(h, salt)
	hash := fmt.Sprintf("%x", h.Sum(nil))

	// Struct
	return &User{
		Id: getUuid(),
		Username: username,
		PasswordHash: hash,
		PasswordSalt: salt,
		IsAdmin: isAdmin,
		IsRequester: isRequester,
		IsApprover: isApprover,
	}
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

	// Existing user
	existing := u.GetUser(username)
	if existing != nil {
		return nil, newErr(fmt.Sprintf("Username '%s' already taken", username))
	}

	// Create struct
	user := NewUser(username, password, isAdmin, isRequester, isApprover)

	// @todo Save in cluster (not async)

	// Done
	return user, nil
}

// Get user
func (u *UserHandler) GetUser(username string) *User {
	// Fetch
	entry := u.GetUserData(username)
	if entry == nil {
		if username == DEFAULT_ADMIN_USR {
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
	k := fmt.Sprintf("user~%s", username)
	e, _ := datastore.GetEntry(k)
	return e
}