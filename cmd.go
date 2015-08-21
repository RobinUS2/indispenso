package main

import (
	"code.google.com/p/go-uuid/uuid"
)

// @author Robin Verlangen

type Cmd struct {
	Command string
	Pending bool
	Id string
}

func newCmd(command string) *Cmd {
	return &Cmd{
		Id : uuid.New(),
		Command: command,
		Pending: true,
	}
}