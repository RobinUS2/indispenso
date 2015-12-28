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

	assert.Equal(t, "/tmp", c.GetHome())
}

func TestHomeAsRoot(t *testing.T) {
	c := newConfig()
	c.Home = "/"

	assert.Equal(t, "/", c.GetHome())
}

func TestAutoTag(t *testing.T) {
	c := newConfig()
	c.Hostname = "cb01.test.localdomain"
	tags := c.GetTags()
	assert.Len(t, tags, 2)
	assert.NotContains(t, tags, "cb01")
	assert.NotContains(t, tags, "cb")
	assert.Contains(t, tags, "test")
	assert.Contains(t, tags, "localdomain")
	assert.NotContains(t, tags, "")
}

func TestTagsShouldContainsAutoAndRegularTags(t *testing.T) {
	c := newConfig()
	c.TagsList = []string{"test1", "test2"}
	c.Hostname = "localtest"

	tags := c.GetTags()

	assert.Len(t, tags, 3)
	assert.Contains(t, tags, "localtest")
	assert.Contains(t, tags, "test1")
	assert.Contains(t, tags, "test2")
}
