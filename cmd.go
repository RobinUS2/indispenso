package main

import (
	"code.google.com/p/go-uuid/uuid"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

// @author Robin Verlangen

type Cmd struct {
	Command string
	Pending bool
	Id string
	Timeout int // in seconds
}

// Execute
func (c *Cmd) Execute() {
	log.Printf("Executing %s: %s", c.Id, c.Command)

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
	cmd.Wait()
	log.Printf("Finished %s", c.Id)

	// Remove file
	os.Remove(tmpFileName)
}

func newCmd(command string, timeout int) *Cmd {
	return &Cmd{
		Id : uuid.New(),
		Command: command,
		Pending: true,
		Timeout: timeout,
	}
}