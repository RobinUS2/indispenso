package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"os"
	"os/exec"
)

// @author Robin Verlangen

type Cmd struct {
	Command   string
	Pending   bool
	Id        string
	Signature string // makes this only valid from the server to the client based on the preshared token and this is a signature with the command and id
	Timeout   int    // in seconds
}

// Sign the command on the server
func (c *Cmd) Sign(client *RegisteredClient) {
	c.Signature = c.ComputeHmac(client.AuthToken)
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

	// File contents
	var fileBytes bytes.Buffer
	fileBytes.WriteString("#!/bin/bash\n")
	fileBytes.WriteString(c.Command)

	// Write tmp file
	tmpFileName := fmt.Sprintf("/tmp/indispenso_%s", c.Id)
	ioutil.WriteFile(tmpFileName, fileBytes.Bytes(), 0644)

	// Run file
	cmd := exec.Command("bash", tmpFileName)
	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to run command: %s", err)
	}

	// Wait for completion
	waitE := cmd.Wait()
	if waitE != nil {
		log.Printf("Failed to wait for exit of command: %s", waitE)
	}
	log.Printf("Finished %s", c.Id)

	// Remove file
	os.Remove(tmpFileName)
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
