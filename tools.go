// @author Robin Verlangen
// Discovery service used to detect cluster

package main

import (
	"fmt"
	"log"
	"net"
)

func getPulicIp(hostname string) string {
	if len(hostname) == 0 {
		return ""
	}
	var ipAddr string = ""
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		log.Println(fmt.Sprintf("ERR: Failed to resolve ip address %s", err))
	} else {
		for _, a := range addrs {
			if debug {
				log.Println(fmt.Sprintf("DEBUG: Host %s resolves to %s", hostname, a))
			}
			ipAddr = a
		}
	}
	return ipAddr
}
