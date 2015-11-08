package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/antonholmquist/jason"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client methods (one per "slave", communicates with the server)

type Client struct {
	Id                        string
	Hostname                  string
	AuthToken                 string
	ConnectedServerInstanceId string // ID of the server to which it is connected
	mux                       sync.RWMutex
}

// Start client
func (s *Client) Start() bool {
	log.Printf("Starting client %s from seed %s with tags %v", s.Id, conf.Seed, conf.tags)

	// Ping server to register
	s.PingServer()

	// Get auth token from server
	s.AuthServer()

	// Is the client enabled?
	if clientPort != -1 {
		// Start webserver
		go func() {
			log.Printf("Starting client server %s", s.Id)
			router := httprouter.New()
			router.GET("/ping", Ping)

			log.Printf("Failed to start client server %s %v", s.Id, http.ListenAndServe(fmt.Sprintf(":%d", clientPort), router))

		}()
	}

	// Register with server
	go func() {
		c := time.Tick(time.Duration(CLIENT_PING_INTERVAL) * time.Second)
		for _ = range c {
			s.PingServer()

			// Should we reload auth?
			var reloadAuth bool = false
			s.mux.RLock()
			if len(s.AuthToken) < 1 {
				reloadAuth = true
			}
			s.mux.RUnlock()
			if reloadAuth {
				s.AuthServer()
			}
		}
	}()

	// Long poll commands
	go func() {
		for {
			s.PollCmds()
		}
	}()

	return true
}

// Fetch commands
func (s *Client) PollCmds() {
	bytes, err := s._get(fmt.Sprintf("client/%s/cmds", url.QueryEscape(s.Id)))
	if err == nil {
		obj, jerr := jason.NewObjectFromBytes(bytes)
		if jerr == nil {
			status, statusE := obj.GetString("status")
			// Re-auth
			if statusE != nil || status != "OK" {
				log.Println(string(bytes))
				log.Println("Re-authenticate with server")
				s.AuthServer()
				return
			}

			// List commands
			cmds, _ := obj.GetObjectArray("cmds")
			for _, cmd := range cmds {
				id, _ := cmd.GetString("Id")
				command, _ := cmd.GetString("Command")
				signature, _ := cmd.GetString("Signature")
				templateId, _ := cmd.GetString("TemplateId")
				timeout, _ := cmd.GetInt64("Timeout")
				cmd := newCmd(command, int(timeout))
				cmd.ClientId = client.Id
				cmd.TemplateId = templateId
				cmd.Id = id
				cmd.Signature = signature
				cmd.Execute(s)
			}
		}
	} else {
		// In case of fast error back off a bit
		time.Sleep(1 * time.Second)
	}
}

// Auth server, token is used for verifying commands
func (s *Client) AuthServer() {
	b, e := s._req("POST", fmt.Sprintf("client/%s/auth", url.QueryEscape(s.Id)), nil)
	if e == nil {
		obj, jerr := jason.NewObjectFromBytes(b)
		if jerr == nil {
			// Get signature
			token, et := obj.GetString("token")
			if et != nil || len(token) < 1 {
				return
			}

			// Get token signatur
			tokenSignature, ets := obj.GetString("token_signature")
			if ets != nil || len(tokenSignature) < 1 {
				return
			}

			// Verify token signature with our secure token
			hasher := sha256.New()
			hasher.Write([]byte(token))
			hasher.Write([]byte(conf.SecureToken))
			expectedTokenSignature := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			// The same?
			if tokenSignature != expectedTokenSignature {
				log.Println("ERROR! Token signature from server is invalid, communication between server and client might be tampered with")
				return
			}

			// Store token if it is valid
			s.mux.Lock()
			s.AuthToken = token
			s.mux.Unlock()
			log.Printf("Client authenticated with server")
		}
	}
}

// Ping server
func (s *Client) PingServer() {
	bytes, e := s._get(fmt.Sprintf("client/%s/ping?tags=%s&hostname=%s", url.QueryEscape(s.Id), url.QueryEscape(strings.Join(conf.Tags(), ",")), url.QueryEscape(s.Hostname)))
	if e == nil {
		obj, jerr := jason.NewObjectFromBytes(bytes)
		if jerr == nil {
			status, statusE := obj.GetString("status")
			serverInstanceId, _ := obj.GetString("server_instance_id")

			// Ping failed, re-authenticate
			if statusE != nil || status != "OK" {
				log.Println(string(bytes))
				log.Println("Re-authenticate with server")
				s.AuthServer()
			} else {
				// Only log a connect if the instance ID changed
				if len(s.ConnectedServerInstanceId) == 0 || s.ConnectedServerInstanceId != serverInstanceId {
					s.ConnectedServerInstanceId = serverInstanceId
					log.Println(fmt.Sprintf("Client registered with server %s", s.ConnectedServerInstanceId))
				}
			}
		}
	}
}

// Get
func (s *Client) _get(uri string) ([]byte, error) {
	return s._req("GET", uri, make([]byte, 0))
}

// Generic request method with retry handling
func (s *Client) _req(method string, uri string, data []byte) ([]byte, error) {
	var bytes []byte = nil
	var err error = nil
	for i := 0; i < 10; i++ {
		bytes, err = s._reqUnsafe(method, uri, data)
		if err == nil && bytes != nil && len(bytes) > 0 {
			return bytes, err
		}

		// Sleep a bit before the retry and apply ~25ms jitter
		var sleep float64 = 25 + float64(rand.Intn(50)) + (math.Pow(float64(i), 2) * 10000)
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}
	if err != nil {
		log.Printf("Failed request after retries to %s with error: %s", uri, err)
	}
	return bytes, err
}

// Generic request method
func (s *Client) _reqUnsafe(method string, uri string, data []byte) ([]byte, error) {
	// Transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		}, // Ignore certificate as this is self generated and invalid
	}
	// For some reasons connections were not closed, this helps
	defer tr.CloseIdleConnections()

	// Client
	client := &http.Client{
		Transport: tr,
	}

	// Sanitize urls
	uri = fmt.Sprintf("/%s", strings.TrimLeft(uri, "/"))

	// Append random string to uri
	var randStr, _ = secureRandomString(32)
	if !strings.Contains(uri, "?") {
		uri = fmt.Sprintf("%s?_rand=%s", uri, randStr)
	} else {
		uri = fmt.Sprintf("%s&_rand=%s", uri, randStr)
	}
	url := fmt.Sprintf("%s%s", strings.TrimRight(seedUri, "/"), uri)

	// Log
	if debug {
		log.Printf("%s %s (req bytes %d)", method, url, len(data))
	}

	// Req
	var buf *bytes.Buffer
	if data != nil && len(data) > 0 {
		buf = bytes.NewBuffer(data)
	} else {
		buf = bytes.NewBuffer(make([]byte, 0))
	}
	req, reqErr := http.NewRequest(method, url, buf)
	if reqErr != nil {
		return nil, reqErr
	}

	// Signed token
	hasher := sha256.New()
	hasher.Write([]byte(uri))
	hasher.Write([]byte(conf.SecureToken))
	signedToken := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	// Auth token
	req.Header.Add("X-Auth", signedToken)

	// Execute
	resp, respErr := client.Do(req)
	if respErr != nil {
		return nil, respErr
	}

	// Read body
	defer resp.Body.Close()
	body, bodyErr := ioutil.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}
	return body, nil
}

// Create new client
func newClient() *Client {
	return &Client{
		Id:       hostname,
		Hostname: hostname,
	}
}
