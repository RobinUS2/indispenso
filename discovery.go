// @author Robin Verlangen
// Discovery service used to detect cluster

package main

// Imports
import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

// Discovery constants
const PING_TIMEOUT = 30 * time.Second

// Node (entity in the Dispenso cluster)
type Node struct {
	Host string // Fully qualified hostname
	Port int    // Port on which Dispenso runs
}

// Full display name
func (n *Node) FullName() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

// Ping a node
func (n *Node) Ping() bool {
	conn, err := net.DialTimeout("tcp", n.FullName(), PING_TIMEOUT)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Message (payload transmitted between nodes containing instructions)
type Message struct {
	Type    MessageType // Type of message
	Payload string      // JSON payload
}

// Message types, enum-like datastructure, use "MessageType" as wrapper
type MessageType struct {
	code messageType
}
type messageType int

const (
	discoveryPing     messageType = iota + 1 // Initial discovery ping
	disocveryResponse                        // Discovery response
	discoveryMeta                            // Metadata beyond initial discovery
	configuration                            // Used to update configuration in the cluster
	taskRequest                              // New task submission
	taskApproval                             // Approve task
	taskReject                               // Reject task
	taskExecution                            // After being approved a task execution will be sent to the nodes
)

// Discovery service
type DiscoveryService struct {
	Nodes []*Node // List of nodes
}

// Create discovery service
func NewDiscoveryService() *DiscoveryService {
	return &DiscoveryService{}
}

// Set seeds
func (d *DiscoveryService) SetSeeds(seeds []string) error {
	for _, seed := range seeds {
		// Simple seed validation
		split := strings.Split(seed, ":")
		var port int = defaultPort
		if len(split) > 2 {
			log.Println(fmt.Sprintf("ERROR: Seed %s host:port format invalid", seed))
			continue
		} else if len(split) == 1 {
			// Default port
		} else {
			// User port
			var err error
			port, err = strconv.Atoi(split[1])
			if err != nil {
				log.Println(fmt.Sprintf("ERROR: Seed %s port format invalid", seed))
				continue
			}
		}

		// Add node
		n := &Node{
			Host: split[0],
			Port: port,
		}
		d.Nodes = append(d.Nodes, n)
	}
	return nil
}

// Run discovery service
func (d *DiscoveryService) Start() {
	go func() {
		log.Println("INFO: Starting discovery")

		// Iterate nodes
		for _, node := range d.Nodes {
			if node.Ping() {
				log.Println(fmt.Sprintf("INFO: Detected %s", node.FullName()))
			} else {
				log.Println(fmt.Sprintf("WARN: Failed to detect %s", node.FullName()))
			}
		}

		// @todo Run every once in a while, and remove shutdown

		shutdown <- true
	}()
}
