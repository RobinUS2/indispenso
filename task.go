// @author Robin Verlangen
// API handler

package main

// Imports
import (
	"log"
	"fmt"
)

// Task struct
type Task struct {
	Targets  []*Node // List of target nodes
	Commands []string // List of commands to execute in that order
}

// Execute task
func (t *Task) Execute() string {
	log.Println(fmt.Sprintf("INFO: Executing task with %d command(s) on %d node(s)", len(t.Commands), len(t.Targets)))
	return "-1"
}

// New task
func NewTask() *Task {
	return &Task{
		Targets:  nil,
		Commands: make([]string, 0),
	}
}
