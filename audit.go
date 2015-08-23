package main

// @author Robin Verlangen
// Audit log

var audit *Audit = newAudit()

type Audit struct {
}

func (a *Audit) Log(usr *User, msg string) {
	log.Printf("%s %s", usr.Username, msg)
}

func newAudit() *Audit {
	return &Audit{}
}
