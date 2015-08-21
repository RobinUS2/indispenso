package main

import (
	"code.google.com/p/go-uuid/uuid"
)

// @author Robin Verlangen

type Cmd struct {
	Command string
	Pending bool
	Id string
	Timeout int // in seconds
}

func newCmd(command string, timeout int) *Cmd {
	return &Cmd{
		Id : uuid.New(),
		Command: command,
		Pending: true,
		Timeout: timeout,
	}
}