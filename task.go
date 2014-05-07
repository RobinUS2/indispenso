// @author Robin Verlangen
// API handler

package main

// Imports
import (
	"fmt"
	"log"
)

// Task struct
type Task struct {
	Id string // Unique task id
	Targets  []*Node  // List of target nodes
	Commands []string // List of commands to execute in that order
}

// Execute node
func (t *Task) executeNode(node *Node) bool {
	return false
}

// Execute task
func (t *Task) Execute() string {
	log.Println(fmt.Sprintf("INFO: Executing task %s with %d command(s) on %d node(s)", t.Id, len(t.Commands), len(t.Targets)))
	for _,node := range t.Targets {
		if t.executeNode(node) == false {
			log.Println(fmt.Sprintf("ERROR: Failed executing task %s on %s", t.Id, node.FullName()))
		}
	}
	return t.Id
}

// New task
func NewTask() *Task {
	return &Task{
		Id: getUuid(),
		Targets:  nil,
		Commands: make([]string, 0),
	}
}