package main

// @author Robin Verlangen
// Audit log

var audit *Audit = newAudit()

type Audit struct {
}

func (a *Audit) Log(usr *User, title string, msg string) {
	username := ""
	if usr != nil {
		username = usr.Username
	}
	log.Printf("%s %s %s", username, title, msg)
}

func newAudit() *Audit {
	return &Audit{}
}
