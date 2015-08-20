package main

// Server methods (you probably only need one or two in HA failover mode)

type Server struct {

}

// Start server
func (s *Server) Start() bool {
	// @todo
	return true
}

// Create new server
func newServer() *Server {
	return &Server{}
}