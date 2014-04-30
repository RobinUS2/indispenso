// @author Robin Verlangen
// Main function of Dispenso ( https://github.com/RobinUS2/dispenso )

package main

// Imports
import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

// Constants
const defaultPort int = 8011

// Configuration
var seedNodes string
var serverPort int
var storeState bool
var hostname string
var ipAddr string
var debug bool
var ipv6 bool
var noBindLocalhost bool

// Signal channels
var shutdown chan bool = make(chan bool)

// Set configuration from flags
func init() {
	flag.StringVar(&seedNodes, "seeds", "", "Seed nodes, comma separated host:port tuples (e.g. 12.34.56.78,23.34.45.56:8080")
	flag.IntVar(&serverPort, "port", defaultPort, fmt.Sprintf("Port to bind on (defaults to %d)", defaultPort))
	flag.BoolVar(&storeState, "store-state", true, "Allow to store cluster state on this node (default=true)")
	flag.StringVar(&hostname, "hostname", "", "Hostname (defaults to auto-resolve)")
	flag.StringVar(&ipAddr, "ipaddr", "", "Ip address (defaults to auto-resoolve)")
	flag.BoolVar(&debug, "debug", true, "Debug logging")
	flag.BoolVar(&ipv6, "ipv6", false, "Enable ipv6")
	flag.BoolVar(&noBindLocalhost, "no-bind-localhost", true, "Do not bind localhost")
	flag.Parse()
}

// Main function of dispenso
func main() {
	log.Println(fmt.Sprintf("INFO: Starting dispenso"))

	// Hostname resolution?
	if len(hostname) == 0 {
		var err error
		hostname, err = os.Hostname()
		if err != nil {
			log.Println(fmt.Sprintf("ERR: Failed to resolve hostname %s", err))
		}
	}

	// IP resolution
	if len(ipAddr) == 0 {
		ipAddr = getPulicIp(hostname)
	}

	// Debug log startup
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Hostname %s", hostname))
		log.Println(fmt.Sprintf("DEBUG: IP address %s", ipAddr))
	}

	// Start discovery
	var disco *DiscoveryService = NewDiscoveryService()
	if len(strings.TrimSpace(seedNodes)) > 0 {
		seeds := strings.Split(seedNodes, ",")
		seeds = append(seeds, hostname)
		disco.SetSeeds(seeds)
	}
	disco.Start()

	// Start server
	var server *Server = NewServer()
	server.Start()

	// Wait for shutdown
	<-shutdown
}
