// @author Robin Verlangen
// Datastore

package main

// Datastore
type Datastore struct {
	folder          string // Persistent location
	mutationChannel chan *DatastoreMutation // Channel that acts as buffer for mutations, guarantees order
}

// Mutation
type DatastoreMutation struct {
	key       string // Data key
	value     string // New value
	timestamp int64  // Timestamp when the change request it was issued
}

// Create discovery service
func NewDatastore(persistentLocation string) *Datastore {
	return &Datastore{
		folder:          persistentLocation,
		mutationChannel: make(chan *DatastoreMutation, 10000),
	}
}

// Push mutation into queue to be processed
func (s *Datastore) PushMutation(m *DatastoreMutation) bool {
	s.mutationChannel <- m
	return true
}

// Create new mutation
func (s *Datastore) CreateMutation() *DatastoreMutation {
	return &DatastoreMutation{}
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