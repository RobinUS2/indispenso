// @author Robin Verlangen
// Main function of Indispenso ( https://github.com/RobinUS2/indispenso )

package main

// Imports
import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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
var trace bool
var ipv6 bool
var noBindLocalhost bool
var secretKey []byte = []byte("my_super_secret_string") // @todo define / generate on startup
var discoveryService *DiscoveryService

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
	flag.BoolVar(&trace, "trace", false, "Trace logging")
	flag.BoolVar(&ipv6, "ipv6", false, "Enable ipv6")
	flag.BoolVar(&noBindLocalhost, "no-bind-localhost", true, "Do not bind localhost")
	flag.Parse()
}

// Main function of dispenso
func main() {
	log.Println(fmt.Sprintf("INFO: Starting indispenso"))

	// Interrupt handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			log.Println(fmt.Sprintf("INFO: Shutting down indispenso"))

			// Notify leave
			if discoveryService != nil {
				discoveryService.NotifyLeave()
			}

			os.Exit(1)
		}
	}()

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

	// Init discovery
	discoveryService = NewDiscoveryService()

	// Parse seeds
	seeds := strings.Split(seedNodes, ",")

	// Add localhost as seed
	seeds = append(seeds, fmt.Sprintf("%s:%d", hostname, serverPort))

	// Start discovery
	discoveryService.SetSeeds(seeds)
	discoveryService.Start()

	// Start server
	var server *Server = NewServer()
	server.Start()

	// Wait for shutdown
	<-shutdown
}
