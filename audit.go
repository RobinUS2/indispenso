package main

// @author Robin Verlangen
// Audit log

var audit *Audit = newAudit()

type Audit struct {
}

func (a *Audit) Log(usr *User, title string, msg string) {
	log.Printf("%s %s %s", usr.Username, title, msg)
}

func newAudit() *Audit {
	return &Audit{}
}
