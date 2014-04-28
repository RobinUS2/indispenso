// @author Robin Verlangen
// Discovery service used to detect cluster

package main

// Node (entity in the Dispenso cluster)
type Node struct {
	Host string // Fully qualified hostname
	Port int // Port on which Dispenso runs
}