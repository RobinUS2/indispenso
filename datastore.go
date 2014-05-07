// @author Robin Verlangen
// Datastore

package main

// Imports
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// Constants
const MEM_ENTRY_MUX_BUCKETS int = 128
const FLUSH_INTERVAL = 10 * time.Second
const DEFAULT_INCREMENT = 1

// Datastore
type Datastore struct {
	folder          string                  // Persistent location
	mutationChannel chan *DatastoreMutation // Channel that acts as buffer for mutations, guarantees order

	memTable    map[string]*MemEntry // Datastore in-memory values
	memTableMux sync.RWMutex         // Mutex for outer memtable (e.g. appending / reading)

	memEntryMuxes map[int]*sync.Mutex // Mutex buckets for entries

	walFile     *os.File // Write ahead log file pointer
	walFilename string   // Name of the write ahead log file

	dataFile     *os.File     // Data file pointer
	dataFileLock sync.RWMutex // Lock for data file
	dataFilename string       // Name of the data file (persisted)

	mutatorStarted bool         // Is the mutator started?
	flusherStarted bool         // Is the disk flusher started?
	globalMux      sync.RWMutex // Global mutex for datastore struct values (thus NOT data mutations)
}

// Mem entries
type MemEntry struct {
	Key       string // Data key
	Value     string // New value
	Modified  int64  // Timestamp when last changed
	MuxBucket int    // Bucket of where to find my lock
	IsDeleted bool   // Is this entry deleted? @todo Compact (permantenly remove deleted items)
	// @todo Support TTL
}

// Mutation
type DatastoreMutation struct {
	Key          string // Data key
	Value        string // New value
	Timestamp    int64  // Timestamp when the change request it was issued
	Replicated   bool   // Is already replicated to all nodes?
	MutationMode int    // Mutation type (1 = overwrite, 2 = append, 3 = delete, 4 = increment)
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
		// Skip if this is a delete request
		if m.MutationMode == 3 {
			if debug {
				log.Println(fmt.Sprintf("DEBUG: Dropping delete of non-existent key '%s'", m.Key))
			}
			return pos
		}

		// Default value for counters
		if m.MutationMode == 4 {
			m.Value = fmt.Sprintf("%d", DEFAULT_INCREMENT)
		}

		// Add if this is not a delete request
		s.memTableMux.Lock()
		s.memTable[m.Key] = &MemEntry{
			Key:       m.Key,
			Value:     m.Value,
			Modified:  time.Now().UnixNano(),
			MuxBucket: pos % MEM_ENTRY_MUX_BUCKETS,
		}

		// Increment pos for buckets
		pos++

		if trace {
			log.Println(fmt.Sprintf("TRACE: Create new entry in mux bucket %d", s.memTable[m.Key].MuxBucket))
		}
		v = s.memTable[m.Key]
		s.memTableMux.Unlock()
	} else {
		// Is my update newer than the actual current value?
		if m.Timestamp < v.Modified {
			// Mutation is older than last update, skip
			if debug {
				log.Println(fmt.Sprintf("DEBUG: Dropping old update of key '%s' with timestamp %d", v.Key, v.Modified))
			}
			return pos
		}

		// Lock mux for this bucket
		s.memEntryMuxes[v.MuxBucket].Lock()

		// Update value and timestamp
		if m.MutationMode == 1 {
			// Overwrite
			v.Value = m.Value
			v.IsDeleted = false
		} else if m.MutationMode == 2 {
			// Append
			v.Value = fmt.Sprintf("%s%s", v.Value, m.Value)
			v.IsDeleted = false
		} else if m.MutationMode == 3 {
			// Delete
			v.Value = ""
			v.IsDeleted = true
		} else if m.MutationMode == 4 {
			// Increment
			curVal := int64(0)
			if len(v.Value) > 0 {
				// Try to convert to integer
				newVal, convErr := strconv.ParseInt(v.Value, 10, 64)
				if convErr == nil {
					// Good increment value found
					curVal = newVal
				}
			}
			// Get increment value
			if len(m.Value) == 0 {
				m.Value = fmt.Sprintf("%d", DEFAULT_INCREMENT)
			}
			incVal, incConvErr := strconv.ParseInt(m.Value, 10, 64)
			if incConvErr != nil {
				// Bad increment value found, use default
				incVal = int64(DEFAULT_INCREMENT)
			}
			totVal := curVal + incVal
			v.Value = fmt.Sprintf("%d", totVal)
			if trace {
				log.Println(fmt.Sprintf("TRACE: Incrementing key '%s' from %d with %d to %d", v.Key, curVal, incVal, totVal))
			}
			v.IsDeleted = false
		} else {
			log.Println(fmt.Sprintf("ERR: Dropping unknown mutation message with mode %d", m.MutationMode))
			return pos
		}
		v.Modified = time.Now().UnixNano()
		if trace {
			log.Println(fmt.Sprintf("TRACE: Update value to '%s' in mux bucket %d", v.Value, v.MuxBucket))
		}

		// Unlock bucket
		s.memEntryMuxes[v.MuxBucket].Unlock()
	}

	// Replication
	if m.Replicated == false {
		m.Replicate(false, v.Value)
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
func (m *DatastoreMutation) Replicate(async bool, postMutationValue string) bool {

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

			// Skip nodes that are not connected
			if node.connected == false {
				if trace {
					log.Println(fmt.Sprintf("DEBUG: Drop replication request to %s:%d as connection is lost", ipAddr, serverPort))
				}
				return
			}

			// Mutation
			mutation := getEmptyMetaMsg("data_replication")
			mutation["k"] = m.Key
			mutation["v"] = postMutationValue
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
		walFilename:     fmt.Sprintf("%s.wal_%s_%d.log", persistentLocation, hostname, serverPort),
		dataFilename:    fmt.Sprintf("%s.data_%s_%d.json", persistentLocation, hostname, serverPort),
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
	return &DatastoreMutation{
		MutationMode: 1, // By default overwrite
	}
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

// Open file handle
func (s *Datastore) openFileHandle() bool {
	var fErr error
	s.dataFile, fErr = os.OpenFile(s.dataFilename, os.O_RDWR|os.O_CREATE, 0666)
	if fErr != nil {
		log.Fatal(fmt.Sprintf("ERR: Failed to open data file: %s", fErr))
		return false
	}
	return true
}

// Repair Datastore
func (s *Datastore) Repair(d *DiscoveryService) bool {
	if debug {
		log.Println("DEBUG: Repairing datastore")
	}
	// Retry a few times
	for i := 0; i < 15; i++ {
		// Wait for discovery
		timer := time.NewTimer(time.Second * 2)
		<-timer.C

		// Iterate seeds to find a healthy one
		seeds := d.GetLiveSeedNodes()
		for _, node := range seeds {
			// Request datastore contents of other node
			datastoreJson, err := node.sendData("data-repair", msgToJson(getEmptyMetaMsg("data-repair")))
			if err != nil {
				log.Println(fmt.Sprintf("ERR: Failed to repair data %s", err))
				continue
			}

			// Decode json
			repairTable := make(map[string]*MemEntry)
			err = json.Unmarshal([]byte(datastoreJson), &repairTable)
			if err != nil {
				log.Println(fmt.Sprintf("ERR: Failed to decode repair data file: %s", err))
			}
			if debug {
				log.Println(fmt.Sprintf("DEBUG: Repairing %d datastore entries from %s", len(repairTable), node.FullName()))
			}

			// Iterate changes
			var mutationCounter int
			s.memTableMux.Lock()
			for _, entry := range repairTable {
				// Existing?
				if s.memTable[entry.Key] == nil {
					// No, add
					s.memTable[entry.Key] = entry
					if trace {
						log.Println(fmt.Sprintf("TRACE: Repairing data with key %s (add)", entry.Key))
					}
					mutationCounter++
				} else {
					// Yes, check timestamps
					if s.memTable[entry.Key].Modified < entry.Modified {
						// This one is newer, overwrite
						s.memTable[entry.Key] = entry
						if trace {
							log.Println(fmt.Sprintf("TRACE: Repairing data with key %s (update)", entry.Key))
						}
						mutationCounter++
					}
				}
			}
			s.memTableMux.Unlock()
			if mutationCounter > 0 {
				s.Flush()
			}
			log.Println(fmt.Sprintf("INFO: Finished datastore repair with %d mutations", mutationCounter))
			return true
		}
	}
	log.Println("INFO: No seed nodes to repair datastore")
	return false
}

// Open Datastore
func (s *Datastore) Open() bool {
	// Folder init
	os.MkdirAll(s.folder, 0777)

	// Init muxes
	for i := 0; i < MEM_ENTRY_MUX_BUCKETS; i++ {
		s.memEntryMuxes[i] = &sync.Mutex{}
	}

	// Open data file
	s.dataFileLock.Lock()
	s.openFileHandle()
	s.dataFileLock.Unlock()

	// Recover data from disk
	var dataFileExists bool = true
	if _, err := os.Stat(s.dataFile.Name()); os.IsNotExist(err) {
		dataFileExists = false
	}
	if dataFileExists {
		dataBytes, readErr := ioutil.ReadFile(s.dataFile.Name())
		if readErr != nil {
			log.Fatal(fmt.Sprintf("ERR: Failed to read data file: %s", readErr))
		}
		if len(dataBytes) > 0 {
			err := json.Unmarshal(dataBytes, &s.memTable)
			if err != nil {
				log.Fatal(fmt.Sprintf("ERR: Failed to decode data file: %s", err))
			}
			if debug {
				log.Println(fmt.Sprintf("DEBUG: Recovered %d datastore entries from disk", len(s.memTable)))
			}
		}
	}

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

	// Start flusher
	s.globalMux.Lock()
	if s.flusherStarted == false {
		s.startFlusher()
		s.flusherStarted = true
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

// Start flusher
func (s *Datastore) startFlusher() bool {
	go func(s *Datastore) {
		ticker := time.NewTicker(FLUSH_INTERVAL)
		for {
			select {
			case <-ticker.C:
				// Flush
				s.Flush()
			case <-shutdown:
				ticker.Stop()
				return
			}
		}
	}(s)

	if debug {
		log.Println(fmt.Sprintf("DEBUG: Started datastore flusher"))
	}

	return true
}

// Put entry
func (s *Datastore) PutEntry(key string, value string) bool {
	mutation := getEmptyMetaMsg("data_mutation")
	mutation["k"] = key
	mutation["v"] = value
	// @todo Use the best available node instead of the first one
	_, err := discoveryService.Nodes[0].sendData("data", msgToJson(mutation))
	if err == nil {
		return true
	}
	return false
}

// Get local entry (read from memtable)
func (s *Datastore) GetLocalEntry(key string) (*MemEntry, error) {
	s.memTableMux.RLock()
	v := s.memTable[key]
	s.memTableMux.RUnlock()
	if v == nil || v.IsDeleted == true {
		return nil, errors.New(fmt.Sprintf("Key %s not found in datastore", key))
	}
	return v, nil
}

// Get entry (read from memtable + other nodes)
func (s *Datastore) GetEntry(key string) (*MemEntry, error) {
	// @todo Implement
	return s.GetLocalEntry(key)
}

// Get mem table json
func (s *Datastore) memTableToJson() string {
	// To Json
	s.globalMux.RLock()
	b, err := json.Marshal(s.memTable)
	s.globalMux.RUnlock()
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to convert datastore memtable to json %s", err))
		return ""
	}

	// To string
	return string(b)
}

// Flush Datastore contents to persistent storage
func (s *Datastore) Flush() bool {
	// Open tmp data file
	var fErr error
	var tmpFile *os.File
	tmpFile, fErr = os.OpenFile(fmt.Sprintf("%s.tmp", s.dataFilename), os.O_RDWR|os.O_CREATE, 0666)
	if fErr != nil {
		log.Fatal(fmt.Sprintf("ERR: Failed to open tmp data file: %s", fErr))
	}

	// Write to disk
	jsonStr := s.memTableToJson()
	tmpFile.WriteString(jsonStr)
	tmpFile.Sync()
	tmpFile.Close()

	// Swap new file with old file
	s.dataFileLock.Lock()
	closeErr := s.dataFile.Close()
	if closeErr != nil {
		log.Println(fmt.Sprintf("ERR: Failed to close data file: %s", closeErr))
		return false
	}
	removeErr := os.Remove(s.dataFile.Name())
	if removeErr != nil {
		log.Println(fmt.Sprintf("ERR: Failed to remove old data file: %s", removeErr))
		return false
	}
	renameErr := os.Rename(tmpFile.Name(), s.dataFile.Name())
	s.openFileHandle()
	s.dataFileLock.Unlock()

	// Remove tmp file
	if renameErr == nil {
		os.Remove(tmpFile.Name())
	} else {
		log.Println(fmt.Sprintf("ERR: Failed to swap tmp data file to data file: %s", renameErr))
	}

	// Sync write ahead log
	s.walFile.Sync()

	// Debug
	if trace {
		log.Println(fmt.Sprintf("TRACE: Flushed datastore"))
	}
	return true
}

// Close Datastore
func (s *Datastore) Close() bool {

	// Close write ahead
	if s.walFile != nil {
		s.walFile.Close()
	}

	// Close data file
	if s.dataFile != nil {
		s.dataFile.Close()
	}

	// @todo Implement
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Closed datastore"))
	}
	return false
}
