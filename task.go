// @author Robin Verlangen
// API handler

package main

// Imports
import (
)

// Task struct
type Task struct {
	Targets []string // List of target nodes
	Commands []string // List of commands to execute in that order
}

// Execute task
func (t *Task) Execute() string {
	// @todo Implement
	return "-1"
}

// New task
func NewTask() *Task {
	return &Task{
		Targets: make([]string, 0),
		Commands: make([]string, 0),
	}
}