// @author Robin Verlangen
// API handler

package main

// Imports
import (
	"encoding/json"
	"fmt"
	"github.com/pmylund/go-cache"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const DATASTORE_TASK_TTL = "86400" // 24 hours

// Task struct
type Task struct {
	Id               string   // Unique task id
	targets          []*Node  // List of target nodes
	Commands         []string // List of commands to execute in that order
	CreatedTimestamp int64    //  Created
}

// Local task (on a specific node)
type LocalTask struct {
	Id       string   // Unique task id
	Commands []string // List of commands to execute in that order
}

// Task discoverer
type TaskDiscoverer struct {
	executionQueue chan *LocalTask // Queue of tasks to be executed
	executionCache *cache.Cache    // Cache of executed task ids
}

// New task discoverer
func NewTaskDiscoverer() *TaskDiscoverer {
	return &TaskDiscoverer{
		executionQueue: make(chan *LocalTask, 1000),
		executionCache: cache.New(60*time.Minute, 30*time.Second),
	}
}

// Task completion key
func GetCompletionKey(taskId string) string {
	return fmt.Sprintf("task~%s~completed~%s", taskId, instanceId)
}

// Task completion key
func GetOutputKey(taskId string) string {
	return fmt.Sprintf("task~%s~output~%s", taskId, instanceId)
}

// Save output
func (lt *LocalTask) SaveOutput(output string) bool {
	return datastore.PutEntry(GetOutputKey(lt.Id), output, DATASTORE_TASK_TTL)
}

// Run local task
func (lt *LocalTask) Run() bool {
	// Task file
	var tmpFolder string = "/tmp" // @todo Configure
	var shell string = "sh"       // @todo Configure
	var tmpFile string = fmt.Sprintf("%s/%s", tmpFolder, lt.Id)
	file, fopenErr := os.OpenFile(tmpFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if fopenErr != nil {
		log.Println(fmt.Sprintf("ERROR: Failed to open tmp task file %s", fopenErr))
		return false
	}
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Opened tmp task file in %s", file.Name()))
	}

	// Print commands to file
	file.WriteString(fmt.Sprintf("#!/bin/%s\n", shell))
	for _, cmd := range lt.Commands {
		file.WriteString(cmd + "\n")
		if debug {
			log.Println(fmt.Sprintf("Writing to '%s': %s", file.Name(), cmd))
		}
	}
	file.Sync()
	file.Close()

	// Execute file
	output, execErr := exec.Command(shell, file.Name()).Output()
	datastore.PutEntry(GetCompletionKey(lt.Id), "1", DATASTORE_TASK_TTL)
	if execErr != nil {
		log.Println(fmt.Sprintf("ERR: Failed to execute tmp task file: %s", execErr))
		return false
	}
	lt.SaveOutput(string(output))
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Task output: %s", output))
	}

	// Cleanup
	removeErr := os.Remove(file.Name())
	if removeErr != nil {
		log.Println(fmt.Sprintf("ERR: Failed to remove tmp task file: %s", removeErr))
		return false
	}
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Removed tmp task file in %s", file.Name()))
	}

	return false
}

// Discover new tasks from queue
func (td *TaskDiscoverer) Discover() bool {
	// Datastore ready?
	if datastore == nil {
		if debug {
			log.Println("DEBUG: Datastore not ready, failed task discovery")
		}
		return false
	}

	// Read list of tasks
	k := fmt.Sprintf("%s~task_ids", instanceId)
	entry, err := datastore.GetEntry(k)
	if err != nil {
		log.Println(fmt.Sprintf("ERROR: Failed to discover tasks: %s", err))
		return false
	}
	if entry == nil {
		// No entry, nothing to do, but that's fine
		return true
	}

	// List of task ids
	taskIds := strings.Split(entry.Value, ",")
	for _, taskId := range taskIds {
		// Skip empty ones
		if len(taskId) == 0 {
			continue
		}

		// Skip imported ones
		_, found := td.executionCache.Get(taskId)
		if found {
			continue
		}

		// Skip completed ones
		doneFlag, _ := datastore.GetEntry(GetCompletionKey(taskId))
		if doneFlag != nil {
			continue
		}

		// Get task meta
		taskData, _ := datastore.GetEntry(fmt.Sprintf("task~%s", taskId))
		if taskData == nil {
			continue
		}

		// Decode
		lt := ReadLocalTask(taskData.Value)
		if lt == nil {
			continue
		}
		// Add to queue + flag as imported
		td.executionCache.Set(lt.Id, "1", 0)
		td.executionQueue <- lt
	}

	// Done
	return true
}

// Start task discoverer
func (td *TaskDiscoverer) Start() bool {
	// Discovery
	go func(td *TaskDiscoverer) {
		ticker := time.NewTicker(200 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				// Discover tasks
				td.Discover()
			case <-shutdown:
				ticker.Stop()
				return
			}
		}
	}(td)

	// Execution
	go func(td *TaskDiscoverer) {
		for {
			var lt *LocalTask
			lt = <-td.executionQueue
			lt.Run()
		}
	}(td)
	return true
}

// Write to datastore
func (t *Task) writeDatastore() bool {
	return datastore.PutEntry(fmt.Sprintf("task~%s", t.Id), t.toJson(), DATASTORE_TASK_TTL)
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
	return datastore.AppendEntry(fmt.Sprintf("%s~task_ids", node.InstanceId), fmt.Sprintf("%s,", t.Id), DATASTORE_TASK_TTL)
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

// Local task
func ReadLocalTask(str string) *LocalTask {
	var lt *LocalTask = &LocalTask{}
	err := json.Unmarshal([]byte(str), &lt)
	if err != nil {
		log.Println(fmt.Sprintf("ERROR: Failed to decode local task json %s", err))
		return nil
	}
	return lt
}

// New task
func NewTask() *Task {
	return &Task{
		Id:       getUuid(),
		targets:  nil,
		Commands: make([]string, 0),
	}
}
