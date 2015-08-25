package main

import (
	// "bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// @author Robin Verlangen

type Cmd struct {
	Command       string
	Pending       bool
	Id            string
	TemplateId    string
	Signature     string // makes this only valid from the server to the client based on the preshared token and this is a signature with the command and id
	Timeout       int    // in seconds
	State         string
	RequestUserId string // User ID of the user that initiated this command
	Created       int64  // Unix timestamp created
	BufOutput     []string
	BufOutputErr  []string
}

// Sign the command on the server
func (c *Cmd) Sign(client *RegisteredClient) {
	c.Signature = c.ComputeHmac(client.AuthToken)
}

// Set local state
func (c *Cmd) SetState(state string) {
	c.State = state
}

// Notify state to server
func (c *Cmd) NotifyServer(state string) {
	// Update local client state
	c.SetState(state)

	// Update server state
	client._req("PUT", fmt.Sprintf("client/%s/cmd/%s/state?state=%s", url.QueryEscape(client.Id), url.QueryEscape(c.Id), url.QueryEscape(state)), nil)
}

// Should we flush the local buffer? After X milliseconds or Y lines
func (c *Cmd) _checkFlushLogs() {
	// At least 10 lines
	if len(c.BufOutput) > 10 || len(c.BufOutputErr) > 10 {
		c._flushLogs()
	}
}

// Write logs to server
func (c *Cmd) _flushLogs() {
	// To JSON
	m := make(map[string][]string)
	m["output"] = c.BufOutput
	m["error"] = c.BufOutputErr
	bytes, je := json.Marshal(m)
	if je != nil {
		log.Printf("Failed to convert logs to JSON: %s", je)
		return
	}

	// Post to server
	uri := fmt.Sprintf("client/%s/cmd/%s/logs", url.QueryEscape(client.Id), url.QueryEscape(c.Id))
	b, e := client._req("PUT", uri, bytes)
	if e != nil || len(b) < 1 {
		log.Printf("Failed log write: %s", e)
	}

	// Clear buffers
	c.BufOutput = make([]string, 0)
	c.BufOutputErr = make([]string, 0)
}

// Log output
func (c *Cmd) LogOutput(line string) {
	// No lock, only one routine can access this

	// Append
	c.BufOutput = append(c.BufOutput, line)

	// Check to flush?
	c._checkFlushLogs()
}

// Log error
func (c *Cmd) LogError(line string) {
	// No lock, only one routine can access this

	// Append
	c.BufOutputErr = append(c.BufOutputErr, line)

	// Check to flush?
	c._checkFlushLogs()
}

// Sign the command
func (c *Cmd) ComputeHmac(token string) string {
	bytes, be := base64.URLEncoding.DecodeString(token)
	if be != nil {
		return ""
	}
	mac := hmac.New(sha256.New, bytes)
	mac.Write([]byte(c.Command))
	mac.Write([]byte(c.Id))
	sum := mac.Sum(nil)
	return base64.URLEncoding.EncodeToString(sum)
}

// Execute command on the client
func (c *Cmd) Execute(client *Client) {
	log.Printf("Executing %s: %s", c.Id, c.Command)

	// Validate HMAC
	if client != nil {
		// Compute mac
		expectedMac := c.ComputeHmac(client.AuthToken)
		if expectedMac != c.Signature || len(c.Signature) < 1 {
			log.Printf("ERROR! Invalid command signature, communication between server and client might be tampered with")
			return
		}
	} else {
		log.Printf("Executing insecure command, unable to validate HMAC of %s", c.Id)
	}

	// Start
	c.NotifyServer("starting")

	// File contents
	var fileBytes bytes.Buffer
	fileBytes.WriteString("#!/bin/bash\n")
	fileBytes.WriteString(c.Command)

	// Write tmp file
	tmpFileName := fmt.Sprintf("/tmp/indispenso_%s", c.Id)
	ioutil.WriteFile(tmpFileName, fileBytes.Bytes(), 0644)

	// Remove file once done
	defer os.Remove(tmpFileName)

	// Run file
	cmd := exec.Command("bash", tmpFileName)
	var out bytes.Buffer
	var outerr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outerr

	// Consume streams
	// go func() {
	// 	p, pe := cmd.StdoutPipe()
	// 	if pe != nil {
	// 		log.Printf("Pipe error: %s", pe)
	// 		return
	// 	}
	// 	scanner := bufio.NewScanner(p)
	// 	for scanner.Scan() {
	// 		txt := scanner.Text()
	// 		c.LogOutput(txt)
	// 		if debug {
	// 			log.Println(scanner.Text())
	// 		}
	// 	}
	// 	if err := scanner.Err(); err != nil {
	// 		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	// 	}
	// }()
	// go func() {
	// 	p, pe := cmd.StderrPipe()
	// 	if pe != nil {
	// 		log.Printf("Pipe error: %s", pe)
	// 		return
	// 	}
	// 	scanner := bufio.NewScanner(p)
	// 	for scanner.Scan() {
	// 		txt := scanner.Text()
	// 		c.LogError(txt)
	// 		if debug {
	// 			log.Println(scanner.Text())
	// 		}
	// 	}
	// 	if err := scanner.Err(); err != nil {
	// 		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	// 	}
	// }()

	// Start
	err := cmd.Start()
	if err != nil {
		c.NotifyServer("failed")
		log.Printf("Failed to start command: %s", err)
		return
	}
	c.NotifyServer("started")

	// Timeout mechanism
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(time.Duration(c.Timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill %s: %s", c.Id, err)
			return
		}
		<-done // allow goroutine to exit
		c.NotifyServer("killed")
		log.Printf("Process %s killed", c.Id)
	case err := <-done:
		if err != nil {
			c.NotifyServer("failed")
			log.Printf("Process %s done with error = %v", c.Id, err)
		} else {
			c.NotifyServer("finished")
			log.Printf("Finished %s", c.Id)
		}
	}

	// Logs
	for _, line := range strings.Split(out.String(), "\n") {
		c.LogOutput(line)
	}
	for _, line := range strings.Split(outerr.String(), "\n") {
		c.LogError(line)
	}
	// Final flush
	c._flushLogs()
}

func newCmd(command string, timeout int) *Cmd {
	// Default timeout if not valid
	if timeout < 1 {
		timeout = DEFAULT_COMMAND_TIMEOUT
	}

	// Id
	id, _ := uuid.NewV4()

	// Create instance
	return &Cmd{
		Id:           id.String(),
		Command:      command,
		Pending:      true,
		Timeout:      timeout,
		Created:      time.Now().Unix(),
		BufOutput:    make([]string, 0),
		BufOutputErr: make([]string, 0),
	}
}
