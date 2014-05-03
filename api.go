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
	uh := NewUserHandler()
	userData := uh.GetUser(fmt.Sprintf("%s", data["username"]))
	resp["user"] = userData
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
