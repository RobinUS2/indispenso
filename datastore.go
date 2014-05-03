// @author Robin Verlangen
// Datastore

package main

// Imports
import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Constants
const MEM_ENTRY_MUX_BUCKETS int = 128

// Datastore
type Datastore struct {
	folder          string                  // Persistent location
	mutationChannel chan *DatastoreMutation // Channel that acts as buffer for mutations, guarantees order

	memTable    map[string]*MemEntry // Datastore in-memory values
	memTableMux sync.RWMutex         // Mutex for outer memtable (e.g. appending / reading)

	memEntryMuxes map[int]*sync.Mutex // Mutex buckets for entries

	walFile     *os.File // Write ahead log file pointer
	walFilename string   // Name of the write ahead log file

	mutatorStarted bool
	globalMux      sync.RWMutex // Global mutex for datastore struct values (thus NOT data mutations)
}

// Mem entries
type MemEntry struct {
	Key       string // Data key
	Value     string // New value
	Modified  int64  // Timestamp when last changed
	muxBucket int    // Bucket of where to find my lock
}

// Mutation
type DatastoreMutation struct {
	Key        string // Data key
	Value      string // New value
	Timestamp  int64  // Timestamp when the change request it was issued
	Replicated bool   // Is already replicated to all nodes?
}

// Execute mutation
func (m *DatastoreMutation) ExecuteMutation(s *Datastore, pos int) int {
	if trace {
		log.Println(fmt.Sprintf("TRACE: Mutation '%s' = '%s'", m.Key, m.Value))
	}

	// Read current value
	s.memTableMux.RLock()
	v, _ := s.GetEntry(m.Key)
	s.memTableMux.RUnlock()

	// Not set?
	if v == nil {
		s.memTableMux.Lock()
		s.memTable[m.Key] = &MemEntry{
			Key:       m.Key,
			Value:     m.Value,
			Modified:  time.Now().UnixNano(),
			muxBucket: pos % MEM_ENTRY_MUX_BUCKETS,
		}

		// Increment pos for buckets
		pos++

		if trace {
			log.Println(fmt.Sprintf("TRACE: Create new entry in mux bucket %d", s.memTable[m.Key].muxBucket))
		}
		s.memTableMux.Unlock()
	} else {
		// Is my update newer than the actual current value?
		if m.Timestamp < v.Modified {
			// Mutation is older than last update, skip
			if debug {
				log.Println(fmt.Sprintf("DEBUG: Dropping old update of key '%s'with timestamp %d", v.Key, v.Modified))
			}
			return pos
		}

		// Lock mux for this bucket
		s.memEntryMuxes[v.muxBucket].Lock()

		// Update value and timestamp
		v.Value = m.Value
		v.Modified = time.Now().UnixNano()
		if trace {
			log.Println(fmt.Sprintf("TRACE: Update value to '%s' in mux bucket %d", v.Value, v.muxBucket))
		}

		// Unlock bucket
		s.memEntryMuxes[v.muxBucket].Unlock()
	}

	// Replication
	if m.Replicated == false {
		m.Replicate(false)
	}

	// Done
	return pos
}

// Perist to disk for recovery
func (m *DatastoreMutation) PersistDisk(async bool) bool {
	// To Json
	b, err := json.Marshal(m)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to convert datastore mutation to json %s", err))
		return false
	}

	// To string
	jsonStr := string(b)

	// Write
	return datastore.WriteMutation(jsonStr, async)
}

// Replicate mutation
func (m *DatastoreMutation) Replicate(async bool) bool {

	// Send to all nodes
	for _, node := range discoveryService.Nodes {
		f := func(node *Node) {
			// Skip ourselves in the replication process
			if node.Addr == ipAddr && node.Port == serverPort {
				if trace {
					log.Println(fmt.Sprintf("DEBUG: Drop local replication request to %s:%d", ipAddr, serverPort))
				}
				return
			}

			// Mutation
			mutation := getEmptyMetaMsg("data_replication")
			mutation["k"] = m.Key
			mutation["v"] = m.Value
			mutation["r"] = "1" // Replication request

			// @todo Validate
			node.sendData("data", msgToJson(mutation))
			// @todo On failure, write to a hinted handoff writes file for replay on startup
		}

		// Execute
		if async {
			go f(node)
		} else {
			f(node)
		}
	}

	return true
}

// Create discovery service
func NewDatastore(persistentLocation string) *Datastore {
	return &Datastore{
		folder:          persistentLocation,
		mutationChannel: make(chan *DatastoreMutation, 10000),
		memTable:        make(map[string]*MemEntry),
		memEntryMuxes:   make(map[int]*sync.Mutex),
		walFilename:     fmt.Sprintf(".wal_%s_%d.log", hostname, serverPort),
	}
}

// Push mutation into queue to be processed
func (s *Datastore) PushMutation(m *DatastoreMutation) bool {
	m.PersistDisk(false)
	s.mutationChannel <- m
	return true
}

// Create new mutation
func (s *Datastore) CreateMutation() *DatastoreMutation {
	return &DatastoreMutation{}
}

// Write mutation to disk
func (s *Datastore) WriteMutation(json string, async bool) bool {
	s.walFile.WriteString(fmt.Sprintf("%s\n", json))
	if async == false {
		// Persist to disk immediately
		s.walFile.Sync()
	}
	return true
}

// Open Datastore
func (s *Datastore) Open() bool {
	// @todo Test folder

	// Init muxes
	for i := 0; i < MEM_ENTRY_MUX_BUCKETS; i++ {
		s.memEntryMuxes[i] = &sync.Mutex{}
	}

	// Recover data from disk

	// Open write ahead log file
	var fErr error
	s.walFile, fErr = os.OpenFile(s.walFilename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if fErr != nil {
		log.Fatal(fmt.Sprintf("ERR: Failed to open write ahead log: %s", fErr))
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
			m = <-s.mutationChannel
			pos = m.ExecuteMutation(s, pos)
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
	return v, nil
}

// Flush Datastore contents to persistent storage
func (s *Datastore) Flush() bool {
	// @todo Implement store data on disk

	// Sync write ahead log
	s.walFile.Sync()

	// Debug
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Flushed datastore"))
	}
	return false
}

// Close Datastore
func (s *Datastore) Close() bool {

	// Close write ahead
	if s.walFile != nil {
		s.walFile.Close()
	}

	// @todo Implement
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Closed datastore"))
	}
	return false
}
