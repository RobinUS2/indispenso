package main

import (
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
	"time"
)

// Client methods (one per "slave", communicates with the server)

type Client struct {
	Id       string
	Hostname string
}

// Start client
func (s *Client) Start() bool {
	log.Println("Starting client")

	// Start webserver
	go func() {
		router := httprouter.New()
		router.GET("/ping", Ping)

		log.Printf("%v", http.ListenAndServe(fmt.Sprintf(":%d", clientPort), router))
	}()

	// Register with server
	go func() {
		go func() {
			s.PingServer()
		}()
		c := time.Tick(time.Duration(CLIENT_PING_INTERVAL) * time.Second)
		for _ = range c {
			s.PingServer()
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
			cmds, _ := obj.GetObjectArray("cmds")
			for _, cmd := range cmds {
				id, _ := cmd.GetString("Id")
				command, _ := cmd.GetString("Command")
				timeout, _ := cmd.GetInt64("Timeout")
				cmd := newCmd(command, int(timeout))
				cmd.Id = id
				cmd.Execute()
			}
		}
	} else {
		// In case of fast error back off a bit
		time.Sleep(1 * time.Second)
	}
}

// Ping server
func (s *Client) PingServer() {
	s._get(fmt.Sprintf("client/%s/ping?tags=%s&hostname=%s", url.QueryEscape(s.Id), url.QueryEscape(strings.Join(conf.Tags(), ",")), url.QueryEscape(s.Hostname)))
}

// Get
func (s *Client) _get(uri string) ([]byte, error) {
	return s._req("GET", uri, nil)
}

// Generic request method with retry handling
func (s *Client) _req(method string, uri string, data []byte) ([]byte, error) {
	var bytes []byte = nil
	var err error = nil
	for i := 0; i < 10; i++ {
		bytes, err = s._reqUnsafe(method, uri, data)
		if err == nil {
			return bytes, err
		}

		// Sleep a bit before the retry and apply ~25ms jitter
		var sleep float64 = 25 + float64(rand.Intn(50)) + (math.Pow(float64(i), 2) * 10000)
		time.Sleep(time.Duration(sleep) * time.Millisecond)
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

	// Req
	// @todo support data
	req, reqErr := http.NewRequest(method, url, nil)
	if reqErr != nil {
		return nil, reqErr
	}

	// Signed token
	hasher := sha256.New()
	hasher.Write([]byte(uri))
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
