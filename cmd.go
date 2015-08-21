package main

// @author Robin Verlangen

type Cmd struct {
	Command string
}

func newCmd(command string) *Cmd {
	return &Cmd{
		Command: command,
	}
}