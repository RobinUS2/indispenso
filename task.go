// @author Robin Verlangen
// API handler

package main

// Imports
import (
	"fmt"
	"log"
	"encoding/json"
)

// Task struct
type Task struct {
	Id       string   // Unique task id
	targets  []*Node  // List of target nodes
	Commands []string // List of commands to execute in that order
}

// Write to datastore
func (t *Task) writeDatastore() bool {
	return datastore.PutEntry(fmt.Sprintf("task~%s", t.Id), t.toJson())
}

// Task to json
func (t *Task) toJson() string {
	b, err := json.Marshal(t)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to format task in json: %s", err))
		return "{}"
	}
	return string(b)
}

// Execute node
func (t *Task) executeNode(node *Node) bool {
	// Require datastore
	if datastore == nil {
		log.Println("ERROR: Datastore not available, unable to execute task")
		return false
	}

	// Append this task ID to that nodes task list, trailing comma
	return datastore.AppendEntry(fmt.Sprintf("%s~task_ids", node.InstanceId), fmt.Sprintf("%s,", t.Id))
}

// Execute task
func (t *Task) Execute() string {
	// Add task
	if t.writeDatastore() == false {
		log.Println(fmt.Sprintf("ERROR: Failed to write task %s to datastore", t.Id))
		return "-1"
	}

	// Add to node queues
	log.Println(fmt.Sprintf("INFO: Executing task %s with %d command(s) on %d node(s)", t.Id, len(t.Commands), len(t.targets)))
	for _, node := range t.targets {
		if t.executeNode(node) == false {
			log.Println(fmt.Sprintf("ERROR: Failed executing task %s on %s", t.Id, node.FullName()))
		}
	}
	return t.Id
}

// New task
func NewTask() *Task {
	return &Task{
		Id:       getUuid(),
		targets:  nil,
		Commands: make([]string, 0),
	}
}
