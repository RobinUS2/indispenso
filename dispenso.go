package main

// Imports
import (
	"log"
	"flag"
)

// Config through flags
var seedNodes string
func init() {
	flag.StringVar(&seedNodes, "seeds", "", "Seed nodes, comma separated host:port tuples (e.g. 12.34.56.78,23.34.45.56:8080")
	flag.Parse()
}

// Main function of dispenso
func main() {
	log.Println("Starting dispenso")
}