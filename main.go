package main

import (
	"flag"
	"log"
)

// @author Robin Verlangen
// Indispenso: Distribute, manage, regulate, arrange. Simple & secure management based on consensus.

var conf *Conf
var isServer *bool
var seedUri *string

func main() {
	conf = newConf()

	// Read flags
	isServer = flag.Bool("server", false, "Should this run the server process")
	seedUri = flag.String("seed", "", "Seed URI")
	flag.Parse()
	log.Printf("%t", *isServer)
	log.Printf("%s", *seedUri)
}