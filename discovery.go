// @author Robin Verlangen
// Discovery service used to detect cluster

package main

// Imports
import (
	"log"
)

// Node (entity in the Dispenso cluster)
type Node struct {
	Host string // Fully qualified hostname
	Port int // Port on which Dispenso runs
}

// Message (payload transmitted between nodes containing instructions)
type Message struct {
	Type MessageType // Type of message
	Payload string // JSON payload
}

// Message types, enum-like datastructure, use "MessageType" as wrapper
type MessageType struct {
	code messageType
}
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

// Discovery service
type DiscoveryService struct {
	Nodes []Node // List of nodes
}

// Create discovery service
func NewDiscoveryService() *DiscoveryService {
	return &DiscoveryService{}
}

// Run discovery service
func (*DiscoveryService) Start() {
	go func() {
		log.Println("Starting discovery")
		// @todo Implement
		shutdown <- true
	}()
}