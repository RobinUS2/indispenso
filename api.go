// @author Robin Verlangen
// API handler

package main

// Imports
import (
	"encoding/json"
	"fmt"
	"github.com/pmylund/go-cache"
	"log"
	"math/rand"
	"time"
)

// Api handler
type ApiHandler struct {
	sessionCache *cache.Cache
}

// New api handler
func NewApiHandler() *ApiHandler {
	return &ApiHandler{
		sessionCache: cache.New(60*time.Minute, 30*time.Second),
	}
}

// Check session
func (a *ApiHandler) checkSession(data map[string]interface{}) bool {
	token := fmt.Sprintf("%s", data["session_token"])
	if len(token) == 0 {
		return false
	}

	// Check
	_, found := a.sessionCache.Get(token)
	if found == false {
		return false
	}

	// All greens
	return true
}

// Get API session token
func (a *ApiHandler) newSessionToken(user *User) string {
	var token string = HashPassword(fmt.Sprintf("%d", rand.Int63()), fmt.Sprintf("%d", time.Now().UnixNano()))
	a.sessionCache.Set(token, user.Id, 0)
	return token
}

// Mirror
func (a *ApiHandler) Mirror(data map[string]interface{}) map[string]interface{} {
	return data
}

// Login
func (a *ApiHandler) Auth(data map[string]interface{}) map[string]interface{} {
	resp := a.initResp()

	// Look for user
	uh := NewUserHandler()
	user := uh.GetUser(fmt.Sprintf("%s", data["username"]))
	if user == nil {
		resp["error"] = "User not found"
		return resp
	}

	// Validate password
	suppliedPwdHash := HashPassword(fmt.Sprintf("%s", data["password"]), user.PasswordSalt)
	if suppliedPwdHash != user.PasswordHash {
		resp["error"] = "User not found"
		return resp
	}

	// Validate two factor
	var token string = ""
	if data["token"] != nil {
		tmp := fmt.Sprintf("%s", data["token"])
		if len(tmp) > 0 {
			token = tmp
		}
	}
	if user.IsValidTwoFactor(token) == false {
		resp["error"] = "User not found"
		return resp
	}

	// OK
	user.PasswordHash = ""
	user.PasswordSalt = ""
	resp["user"] = user
	resp["session_token"] = a.newSessionToken(user)
	return resp
}

// Init response
func (a *ApiHandler) initResp() map[string]interface{} {
	return make(map[string]interface{})
}

// Parse json
func (a *ApiHandler) parseJson(str string) map[string]interface{} {
	var m map[string]interface{}
	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to parse request json %s", err))
		return make(map[string]interface{})
	}
	return m
}
