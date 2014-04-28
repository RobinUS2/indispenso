// @author Robin Verlangen
// Discovery service used to detect cluster

package main

// Node (entity in the Dispenso cluster)
type Node struct {
	Host string // Fully qualified hostname
	Port int // Port on which Dispenso runs
}

// Message (payload transmitted between nodes containing instructions)
type Message struct {
	Type MessageType
}

// Message types
type messageType int
const (
	discoveryPing	messageType = iota+1 // Initial discovery ping
	disocveryResponse // Discovery response
	discoveryMeta // Metadata beyond initial discovery
	configuration // Used to update configuration in the cluster
	taskRequest // New task submission
	taskApproval // Approve task
	taskReject // Reject task
	taskExecution // After being approved a task execution will be sent to the nodes
)
type MessageType struct {
	code messageType
}