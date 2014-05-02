// @author Robin Verlangen
// Datastore

package main

// Datastore
type Datastore struct {
	folder string // Persistent location
}

// Create discovery service
func NewDatastore(persistentLocation string) *Datastore {
	return &Datastore{
		folder: persistentLocation,
	}
}

// Open Datastore
func (s *Datastore) Open() bool {
	// @todo Implement
	return false
}

// Flush Datastore contents to persistent storage
func (s *Datastore) Flush() bool {
	// @todo Implement
	return false
}

// Close Datastore
func (s *Datastore) Close() bool {
	// @todo Implement
	return false
}