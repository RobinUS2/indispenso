package main
// @author Robin Verlangen

// Imports
import (
	"log"
	"flag"
	"fmt"
)

// Constants
const defaultPort int = 8011

// Configuration
var seedNodes string
var serverPort int

// Set configuration from flags
func init() {
	flag.StringVar(&seedNodes, "seeds", "", "Seed nodes, comma separated host:port tuples (e.g. 12.34.56.78,23.34.45.56:8080")
	flag.IntVar(&serverPort, "port", defaultPort, fmt.Sprintf("Port to bind on (defaults to %d)", defaultPort))
	flag.Parse()
}

// Main function of dispenso
func main() {
	log.Println("Starting dispenso")
}