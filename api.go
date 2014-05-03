// @author Robin Verlangen
// API handler

package main

// Imports
import (
	"log"
	"fmt"
	"encoding/json"
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
	return data
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