package main

import (
	"net/http"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/RobinUS2/golang-jresp"
)

// Server methods (you probably only need one or two in HA failover mode)

type Server struct {

}

// Start server
func (s *Server) Start() bool {
	log.Println("Starting server")

	// Start webserver
	go func() {
		router := httprouter.New()
	    router.GET("/ping", Ping)
	    router.GET("/client/ping/:hostname", ClientPing)

	    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", serverPort), router))
    }()
	return true
}

// Ping
func ClientPing(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    log.Printf("Client %s registered", ps.ByName("hostname"))
    jr := jresp.NewJsonResp()
	jr.Set("ack", true)
	jr.OK()
    fmt.Fprint(w, jr.ToString(debug))
}

// Ping
func Ping(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	jr := jresp.NewJsonResp()
	jr.Set("ping", "pong")
	jr.OK()
    fmt.Fprint(w, jr.ToString(debug))
}

// Create new server
func newServer() *Server {
	return &Server{}
}