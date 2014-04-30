// @author Robin Verlangen
// Main function of Dispenso ( https://github.com/RobinUS2/dispenso )

package main

// Imports
import (
	"flag"
	"fmt"
	"log"
	"strings"
)

// Constants
const defaultPort int = 8011

// Configuration
var seedNodes string
var serverPort int
var storeState bool

// Signal channels
var shutdown chan bool = make(chan bool)

// Set configuration from flags
func init() {
	flag.StringVar(&seedNodes, "seeds", "", "Seed nodes, comma separated host:port tuples (e.g. 12.34.56.78,23.34.45.56:8080")
	flag.IntVar(&serverPort, "port", defaultPort, fmt.Sprintf("Port to bind on (defaults to %d)", defaultPort))
	flag.BoolVar(&storeState, "store-state", true, "Allow to store cluster state on this node (default=true)")
	flag.Parse()
}

// Main function of dispenso
func main() {
	log.Println("INFO: Starting dispenso")

	// Start discovery
	var disco *DiscoveryService = NewDiscoveryService()
	if len(strings.TrimSpace(seedNodes)) > 0 {
		disco.SetSeeds(strings.Split(seedNodes, ","))
	}
	disco.Start()

	// Start server
	var server *Server = NewServer()
	server.Start()

	// Wait for shutdown
	<-shutdown
}