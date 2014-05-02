// @author Robin Verlangen
// Datastore

package main

// Imports
import (
	"sync"
	"log"
	"fmt"
	"time"
	"errors"
)

// Constants
const MEM_ENTRY_MUX_BUCKETS int = 128

// Datastore
type Datastore struct {
	folder          string                  // Persistent location
	mutationChannel chan *DatastoreMutation // Channel that acts as buffer for mutations, guarantees order

	memTable map[string] *MemEntry // Datastore in-memory values
	memTableMux sync.RWMutex // Mutex for outer memtable (e.g. appending / reading)

	memEntryMuxes map[int] *sync.Mutex // Mutex buckets for entries

	mutatorStarted bool
	globalMux sync.RWMutex // Global mutex for datastore struct values (thus NOT data mutations)
}

// Mem entries
type MemEntry struct {
	key       string // Data key
	value     string // New value
	modified int64  // Timestamp when last changed
	muxBucket int // Bucket of where to find my lock
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
		memTable: make(map[string] *MemEntry),
		memEntryMuxes: make(map[int] *sync.Mutex),
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
	// @todo Test folder

	// Init muxes
	for i := 0; i < MEM_ENTRY_MUX_BUCKETS; i++ {
		s.memEntryMuxes[i] = &sync.Mutex{}
	}

	// Start mutator
	s.globalMux.Lock()
	if s.mutatorStarted == false {
		s.startMutator()
		s.mutatorStarted = true
	}
	s.globalMux.Unlock()

	if debug {
		log.Println(fmt.Sprintf("DEBUG: Opened datastore"))
	}

	return false
}

// Start mutator
func (s *Datastore) startMutator() bool {
	go func() {
		var pos int = 0
		for {
			// Read mutation from channel
			var m *DatastoreMutation
			m = <- s.mutationChannel
			if trace {
				log.Println(fmt.Sprintf("TRACE: Mutation '%s' = '%s'", m.key, m.value))
			}

			// Read current value
			s.memTableMux.RLock()
			v,_ := s.GetEntry(m.key)
			s.memTableMux.RUnlock()

			// Not set?
			if v == nil {
				s.memTableMux.Lock()
				s.memTable[m.key] = &MemEntry{
					key: m.key,
					value: m.value,
					modified: time.Now().UnixNano(),
					muxBucket: pos % MEM_ENTRY_MUX_BUCKETS,
				}

				// Increment pos for buckets
				pos++

				if trace {
					log.Println(fmt.Sprintf("TRACE: Create new entry in mux bucket %d", s.memTable[m.key].muxBucket))
				}
				s.memTableMux.Unlock()
			} else {
				// Is my update newer than the actual current value?
				if m.timestamp < v.modified {
					// Mutation is older than last update, skip
					continue
				}

				// Lock mux for this bucket
				s.memEntryMuxes[v.muxBucket].Lock()

				// Update value and timestamp
				v.value = m.value
				v.modified = time.Now().UnixNano()
				if trace {
					log.Println(fmt.Sprintf("TRACE: Update value to '%s' in mux bucket %d", v.value, v.muxBucket))
				}

				// Unlock bucket
				s.memEntryMuxes[v.muxBucket].Unlock()
			}
		}
	}()

	if debug {
		log.Println(fmt.Sprintf("DEBUG: Started datastore mutator"))
	}

	return true
}

// Get entry
func (s *Datastore) GetEntry(key string) (*MemEntry, error) {
	s.memTableMux.RLock()
	v := s.memTable[key]
	s.memTableMux.RUnlock()
	if v == nil {
		return nil, errors.New(fmt.Sprintf("Key %s not found in datastore", key))
	}
	return v, nil;
}

// Flush Datastore contents to persistent storage
func (s *Datastore) Flush() bool {
	// @todo Implement
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Flushed datastore"))
	}
	return false
}

// Close Datastore
func (s *Datastore) Close() bool {
	// @todo Implement
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Closed datastore"))
	}
	return false
}
