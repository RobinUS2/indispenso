// @author Robin Verlangen
// API handler

package main

// Imports
import (
	"encoding/json"
	"fmt"
	"log"
)

// Api handler
type ApiHandler struct {
}

// New api handler
func NewApiHandler() *ApiHandler {
	return &ApiHandler{}
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

	// OK
	resp["user"] = user
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
