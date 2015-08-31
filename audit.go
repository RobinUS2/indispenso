package main

// @author Robin Verlangen
// Audit log

import (
	"strings"
)

var audit *Audit = newAudit()

type Audit struct {
}

func (a *Audit) Log(usr *User, title string, msg string) {
	elms := make([]string, 0)
	if usr != nil {
		elms = append(elms, usr.SessionIpAddress)
		elms = append(elms, usr.Username)
	}
	elms = append(elms, title)
	elms = append(elms, msg)
	log.Println(strings.Join(elms, " "))
}

func newAudit() *Audit {
	return &Audit{}
}
