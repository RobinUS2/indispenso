package main

import (
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAutoRepair(t *testing.T) {
	c := newConfig()

	c.EndpointURI = "test-host.localdomain"
	c.AutoRepair()
	assert.Equal(t, "https://test-host.localdomain:"+cast.ToString(c.ServerPort)+"/", c.EndpointURI)

	c.EndpointURI = "test-host.localdomain:8000"
	c.AutoRepair()
	assert.Equal(t, "https://test-host.localdomain:8000/", c.EndpointURI)

	c.EndpointURI = "http://test-host.localdomain:8000/"
	c.AutoRepair()
	assert.Equal(t, "http://test-host.localdomain:8000/", c.EndpointURI)
}

func TestEmptyEndpointUriDuringAutoRepair(t *testing.T) {
	c := newConfig()

	c.EndpointURI = ""
	c.AutoRepair()
	assert.Equal(t, "", c.EndpointURI)
}

func TestHomeWithTrailingSlash(t *testing.T) {
	c := newConfig()
	c.Home = "/tmp/"

	assert.Equal(t,"/tmp",c.GetHome())
}

func TestHomeAsRoot(t *testing.T) {
	c := newConfig()
	c.Home = "/"

	assert.Equal(t,"/",c.GetHome())
}
