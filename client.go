package main

// Client methods (one per "slave", communicates with the server)

type Client struct {

}

// Start c;oemt
func (s *Client) Start() bool {
	// @todo
	return true
}

// Create new client
func newClient() *Client {
	return &Client{}
}