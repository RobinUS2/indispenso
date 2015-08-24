package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"time"
)

// @author Robin Verlangen

type Cmd struct {
	Command   string
	Pending   bool
	Id        string
	Signature string // makes this only valid from the server to the client based on the preshared token and this is a signature with the command and id
	Timeout   int    // in seconds
	State     string
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
}

func newCmd(command string, timeout int) *Cmd {
	id, _ := uuid.NewV4()
	return &Cmd{
		Id:      id.String(),
		Command: command,
		Pending: true,
		Timeout: timeout,
	}
}
