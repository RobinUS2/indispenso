// @author Robin Verlangen
// Discovery service used to detect cluster

package main

// Imports
import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Discovery constants
const PING_TIMEOUT = 30 * time.Second
const PING_INTERVAL = 10 * time.Second

// Node (entity in the Dispenso cluster)
type Node struct {
	DiscoveryService *DiscoveryService // Discovery service reference
	Host             string            // Fully qualified hostname
	Addr             string            // IP address of this node
	Port             int               // Port on which Dispenso runs
	InstanceId       string            // Instance id (unique per startup)

	// @todo Send meta data every once in a while
	metaReceived bool         // Did we receive metadata?
	mux          sync.RWMutex // Locking mechanism
	lastSeen     int64        // Time last seen
}

// Full name
func (n *Node) FullName() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

// Full url
func (n *Node) FullUrl(service string) string {
	return fmt.Sprintf("http://%s/%s", n.FullName(), service)
}

// Redo metadata exchange
func (n *Node) ResetMetaExchanged() bool {
	n.mux.Lock()
	n.metaReceived = false
	n.mux.Unlock()
	return true
}

// Fetch node metadata
func (n *Node) FetchMeta() bool {
	// Fetch data
	var data map[string]string = getEmptyMetaMsg("fetch_meta")
	b := msgToJson(data)
	bodyStr, err := n.sendData("discovery", b)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to request node metadata %s"), err)
		return false
	}
	body := []byte(bodyStr)

	// Parse json
	var f interface{}
	err = json.Unmarshal(body, &f)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to parse node metadata %s"), err)
		return false
	}
	m := f.(map[string]interface{})
	log.Println(fmt.Sprintf("DEBUG: %s", m["time"]))

	// Meta received
	n.mux.Lock()
	n.metaReceived = true
	log.Println(fmt.Sprintf("INFO: Detected %s @ %s", n.FullName(), n.Addr))
	n.mux.Unlock()

	// Exchange meta
	n.ExchangeMeta()

	return true
}

// Get metadata basis
func getEmptyMetaMsg(t string) map[string]string {
	var data map[string]string = make(map[string]string)
	data["ts"] = fmt.Sprintf("%d", time.Now().UnixNano())
	data["msg_id"] = getUuid()
	data["type"] = t
	data["sender"] = hostname
	data["sender_port"] = fmt.Sprintf("%d", serverPort)
	return data
}

// Message to json
func msgToJson(data map[string]string) []byte {
	// To JSON
	b, err := json.Marshal(data)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to format json"))
		return []byte("{}")
	}
	return b
}

// Notify leave
func (n *Node) NotifyLeave() bool {
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Sending leave notification to %s", n.FullName()))
	}

	// Metadata
	var data map[string]string = getEmptyMetaMsg("node_leave")

	// Send data
	b := msgToJson(data)
	_, err := n.sendData("meta", b)
	if err != nil {
		return false
	}

	return true
}

// Exchange node metadata
func (n *Node) ExchangeMeta() bool {
	log.Println("INFO: Exchanging metadata")

	// List nodes
	var nodeStrs []string = make([]string, 0)
	for _, node := range n.DiscoveryService.Nodes {
		if node == nil || len(node.Host) == 0 {
			continue
		}
		nodeStrs = append(nodeStrs, fmt.Sprintf("%s:%d", node.Host, node.Port))
	}

	// Assemble payload
	var data map[string]string = getEmptyMetaMsg("fetch_meta")
	data["nodes"] = strings.Join(nodeStrs, ",")
	b := msgToJson(data)

	// Send data
	_, err := n.sendData("discovery", b)
	if err != nil {
		return false
	}

	return true
}

// Send data
func (n *Node) sendData(endpoint string, b []byte) (string, error) {
	// Debug post
	if debug && b != nil && len(b) > 0 {
		log.Println(fmt.Sprintf("DEBUG: Post data %s", b))
	}

	// Client
	httpclient := &http.Client{}

	// Calculate message digest
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(b)
	signature := mac.Sum(nil)

	// Execute request
	req, reqErr := http.NewRequest("POST", n.FullUrl(endpoint), bytes.NewBufferString(fmt.Sprintf("%s", b)))
	req.Header.Set("User-Agent", "Dispenso")
	headerSig := fmt.Sprintf("sha256=%s", hex.EncodeToString(signature))
	req.Header.Set("X-Message-Digest", headerSig)
	if trace {
		log.Println(fmt.Sprintf("DEBUG: Message digest header: %s", headerSig))
	}
	if reqErr != nil {
		return "", newErr(fmt.Sprintf("Failed request: %s", reqErr))
	}

	// Parse response
	resp, respErr := httpclient.Do(req)
	if respErr != nil {
		return "", newErr(fmt.Sprintf("Failed request: %s", respErr))
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", newErr(fmt.Sprintf("Failed to read response: %s", err))
	}
	respStr := fmt.Sprintf("%s", body)

	// Debug response
	if debug {
		log.Println(fmt.Sprintf("DEBUG: Response data %s", respStr))
	}
	return respStr, nil
}

// Ping a node
func (n *Node) Ping() bool {
	// Knock on the door
	conn, err := net.DialTimeout("tcp", n.FullName(), PING_TIMEOUT)
	if err != nil {
		return false
	}
	conn.Close()

	// Last seen
	n.mux.Lock()
	n.lastSeen = time.Now().UnixNano()
	n.mux.Unlock()

	// Try to fetch metadata
	n.mux.RLock()
	if n.metaReceived == false {
		go func() {
			n.FetchMeta()
		}()
	}
	n.mux.RUnlock()

	// Store last ping in cluster
	mutation := getEmptyMetaMsg("data_mutation")
	mutation["k"] = fmt.Sprintf("ping~%s", n.FullName())
	mutation["v"] = fmt.Sprintf("%d", n.lastSeen)
	n.sendData("data", msgToJson(mutation))

	// OK
	return true
}

// Message (payload transmitted between nodes containing instructions)
type Message struct {
	Type    MessageType // Type of message
	Payload string      // JSON payload
}

// Message types, enum-like datastructure, use "MessageType" as wrapper
type MessageType struct {
	code messageType
}
type messageType int

const (
	discoveryPing     messageType = iota + 1 // Initial discovery ping
	disocveryResponse                        // Discovery response
	discoveryMeta                            // Metadata beyond initial discovery
	configuration                            // Used to update configuration in the cluster
	taskRequest                              // New task submission
	taskApproval                             // Approve task
	taskReject                               // Reject task
	taskExecution                            // After being approved a task execution will be sent to the nodes
)

// Discovery service
type DiscoveryService struct {
	Nodes []*Node // List of nodes

	mux sync.RWMutex // Locking mechanism
}

// Create discovery service
func NewDiscoveryService() *DiscoveryService {
	return &DiscoveryService{}
}

// Create node
func (d *DiscoveryService) NewNode(host string, port int, addr string) *Node {
	return &Node{
		DiscoveryService: d,
		Host:             host,
		Port:             port,
		Addr:             addr,
	}
}

// Add node
func (d *DiscoveryService) AddNode(n *Node) bool {
	// Ensure we have a host
	if len(n.Host) == 0 {
		return false
	}

	// Look for duplicates
	d.mux.RLock()
	for _, node := range d.Nodes {
		if (node.Host == n.Host && node.Port == n.Port) || (node.Addr == n.Addr && node.Port == n.Port) {
			// Match found
			d.mux.RUnlock()
			return false
		}
	}
	d.mux.RUnlock()

	// Append
	d.mux.Lock()
	d.Nodes = append(d.Nodes, n)
	d.mux.Unlock()
	return true
}

// Remove node
func (d *DiscoveryService) RemoveNode(n *Node) bool {
	var i int = -1
	for in, node := range d.Nodes {
		if node == n {
			i = in
			break
		}
	}

	// Found?
	if i == -1 {
		log.Println(fmt.Sprintf("ERROR: Failed to remove host %s", n.FullName()))
		return false
	}

	// Remove
	d.mux.Lock()
	d.Nodes[i] = d.Nodes[len(d.Nodes)-1]
	d.Nodes = d.Nodes[0 : len(d.Nodes)-1]
	log.Println(fmt.Sprintf("INFO: Removed host %s", n.FullName()))
	d.mux.Unlock()

	return true
}

// Set seeds
func (d *DiscoveryService) SetSeeds(seeds []string) error {
	// Add all seeds
	for _, seed := range seeds {
		// Simple seed validation
		split := strings.Split(seed, ":")
		var port int = defaultPort
		if len(split) > 2 {
			log.Println(fmt.Sprintf("ERROR: Seed %s host:port format invalid", seed))
			continue
		} else if len(split) == 1 {
			// Default port
			if split[0] == hostname {
				// Localhost, use port defined
				port = serverPort
			}
		} else {
			// User port
			var err error
			port, err = strconv.Atoi(split[1])
			if err != nil {
				log.Println(fmt.Sprintf("ERROR: Seed %s port format invalid", seed))
				continue
			}
		}

		// Add node
		n := d.NewNode(split[0], port, getPulicIp(split[0]))
		d.AddNode(n)
	}
	return nil
}

// Notify cluster of new node
func (d *DiscoveryService) NotifyJoin() bool {
	for _, node := range d.Nodes {
		if !node.ExchangeMeta() {
			// @todo Keep track of errors and add exponential backoff
			log.Println(fmt.Sprintf("WARN: Failed to exchange metadata %s", node.FullName()))
		}
	}
	return true
}

// Notify leave
func (d *DiscoveryService) NotifyLeave() bool {
	for _, node := range d.Nodes {
		if !node.NotifyLeave() {
			// @todo Keep track of errors and add exponential backoff
			log.Println(fmt.Sprintf("WARN: Failed to notify leave metadata %s", node.FullName()))
		}
	}
	return true
}

// Ping nodes
func (d *DiscoveryService) PingNodes() bool {
	for _, node := range d.Nodes {
		if !node.Ping() {
			// @todo Keep track of errors and add exponential backoff
			node.ResetMetaExchanged()
			log.Println(fmt.Sprintf("WARN: Failed to detect %s", node.FullName()))
		}
	}
	return true
}

// Run discovery service
func (d *DiscoveryService) Start() bool {
	go func() {
		log.Println("INFO: Starting discovery")

		// Iterate nodes
		ticker := time.NewTicker(PING_INTERVAL)
		d.PingNodes()
		for {
			select {
			case <-ticker.C:
				// Discover nodes
				d.PingNodes()
			case <-shutdown:
				ticker.Stop()
				return
			}
		}

		// @todo Run every once in a while, and remove shutdown

		//shutdown <- true
	}()
	return true
}
