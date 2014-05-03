// @author Robin Verlangen
// User handler

package main

// Imports
import (
	"fmt"
)

// User handler
type UserHandler struct {
}

// New user handler
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// Get user data
func (u *UserHandler) GetUser(username string) string {
	k := fmt.Sprintf("user~%s", username)
	e, _ := datastore.GetEntry(k)
	if e == nil {
		// Uups
		return ""
	}
	return e.Value
}