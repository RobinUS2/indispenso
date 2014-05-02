// @author Robin Verlangen
// Tools

package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
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
		for _, ip := range addrs {
			// Skip ipv6 / local ipaddresses
			if isIpv4Ip(ip) == false && ipv6 == false {
				continue
			}
			// Skip local
			if isLocalIp(ip) == true && noBindLocalhost == true {
				continue
			}
			if debug {
				log.Println(fmt.Sprintf("DEBUG: Host %s resolves to %s", hostname, ip))
			}
			ipAddr = ip
		}
	}
	return ipAddr
}

var REGEX_IPV4 *regexp.Regexp = regexp.MustCompile("[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}")
var REGEX_LOCALHOST *regexp.Regexp = regexp.MustCompile("127\\.0\\.0\\.1")

func isIpv4Ip(ip string) bool {
	return REGEX_IPV4.MatchString(ip)
}

func isLocalIp(ip string) bool {
	return REGEX_LOCALHOST.MatchString(ip)
}

func newErr(msg string) error {
	log.Println(fmt.Sprintf("ERR: %s", msg))
	return errors.New(msg)
}
