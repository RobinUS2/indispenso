package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"gopkg.in/ldap.v2"
	"regexp"
	"sync"
)

type LocalUserStore interface {
	AddUser(login string, email string, authType AuthType) (*User, error)
}

type LdapAuthenticator struct {
	conn      *ldap.Conn
	config    *LdapConfig
	userStore LocalUserStore
}

func newLdapAuthenticator(c *LdapConfig, userStore LocalUserStore) *LdapAuthenticator {
	if err := c.Init(); err != nil {
		log.Fatal(fmt.Sprintf("Cannot initialize config %+v due to: %s", c, err))
	}

	a := new(LdapAuthenticator)
	a.config = c
	a.userStore = userStore

	if err := a.Init(); err != nil {
		log.Fatal(fmt.Sprintf("LDAP Init problem: %s", err))
	}

	return a
}

func (a *LdapAuthenticator) Init() error {
	conn, err := a.createConnection()
	if err != nil {
		return fmt.Errorf("Cannot connect to %s due to: %s", a.config.GetAddress(), err)
	}

	if err := conn.Bind(a.config.ManagerDN, a.config.ManagerPassword); err != nil {
		return fmt.Errorf("Cannot bind as user %s due to: %s", a.config.ManagerDN, err)
	}

	a.conn = conn
	return nil
}

func (c *LdapAuthenticator) createConnection() (conn *ldap.Conn, err error) {

	if !c.config.isTLS {
		conn, err = ldap.Dial("tcp", c.config.GetAddress())
		if err != nil {
			return
		}
		err = conn.StartTLS(c.config.tlsConfig)
		return
	} else {
		return ldap.DialTLS("tcp", c.config.GetAddress(), c.config.tlsConfig)
	}
	return conn, nil
}

func (a *LdapAuthenticator) auth(user *User, ar *AuthRequest) (*User, error) {
	if a.conn == nil {
		return nil, errors.New("Cannot authenticate, LdapAuthenticator missconfigured")
	}

	if user != nil && !user.IsAuthType(AUTH_TYPE_LDAP) {
		return nil, errors.New("User doesn't have LDAP auth enabled")
	}

	userEntry, err := a.UserSearch(ar.login)
	if err != nil {
		return nil, fmt.Errorf("User not found in LDAP: %s", err)
	}

	// User found, create connection and try to Bind as it
	userConn, err := a.createConnection()
	if err != nil {
		return nil, errors.New("Cannot create connection to authenticate user")
	}
	defer userConn.Close()

	if err := userConn.Bind(userEntry.DN, ar.credential); err != nil {
		return nil, fmt.Errorf("Authenticating user in LDAP faild due to: %s", err)
	}

	//user not present in user store create new
	if user == nil {
		user, err = a.userStore.AddUser(ar.login, userEntry.GetAttributeValue(a.config.EmailAttr), AUTH_TYPE_LDAP)
		if err != nil {
			return nil, fmt.Errorf("User authenticated by LDAP but cannot be stored in user store due to: %s", err)
		}
	}
	return user, nil
}

func (a *LdapAuthenticator) UserSearch(login string) (*ldap.Entry, error) {
	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		a.config.RootDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		a.config.GetUserSearchFilter(login),
		a.config.GetAttributes(),
		nil,
	)

	sr, err := a.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if len(sr.Entries) < 1 {
		return nil, errors.New("User does not exist")
	}

	if len(sr.Entries) > 1 {
		return nil, errors.New("Too many entries returned")
	}
	return sr.Entries[0], nil
}

type LdapConfig struct {
	ldapConfMux sync.RWMutex
	initialized bool
	isTLS       bool
	host        string
	port        string
	tlsConfig   *tls.Config

	ServerAddress    string
	ManagerDN        string
	ManagerPassword  string
	RootDN           string
	UserSearchBase   string
	UserSearchFilter string
	Attributes       []string
	EmailAttr        string
}

func (c *LdapConfig) GetUserSearchFilter(login string) string {
	return fmt.Sprintf(c.UserSearchFilter, login)
}

func (c *LdapConfig) Init() error {
	if c.initialized {
		//already initialized
		return nil
	}


	c.ldapConfMux.Lock()
	defer c.ldapConfMux.Unlock()

	regExp, _ := regexp.Compile("(ldaps?)://([^/:]+):?([0-9]{0,})?")

	if !regExp.MatchString(c.ServerAddress) {
		return errors.New("Cannot parse srever address")
	}

	matches := regExp.FindAllStringSubmatch(c.ServerAddress, -1)
	c.isTLS = matches[0][1] == "ldaps"
	c.host = matches[0][2]
	c.port = matches[0][3]

	if c.port == "" {
		if c.isTLS {
			c.port = "636"
		} else {
			c.port = "389"
		}
	}

	c.tlsConfig = &tls.Config{InsecureSkipVerify:false,ServerName:c.host}
	c.initialized = true
	return nil
}

func (c *LdapConfig) GetAttributes() []string {
	return append(c.Attributes, c.EmailAttr, "dn")
}

func (c *LdapConfig) GetAddress() string {
	return fmt.Sprintf("%s:%s", c.host, c.port)
}
