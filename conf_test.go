package main

import (
	"github.com/jmcvetta/randutil"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
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

func TestTokenLength(t *testing.T) {
	c := newConfig()
	home, _ := os.Getwd()
	c.Home = home

	var err error

	c.Token, err = randutil.AlphaString(32)
	assert.NoError(t, err)
	assert.NoError(t, c.Validate())

	c.Token, err = randutil.AlphaString(10)
	assert.NoError(t, err)
	assert.Error(t, c.Validate())

}

func TestEmptyTokenValidation(t *testing.T) {
	c := newConfig()
	home, _ := os.Getwd()
	c.Home = home
	c.Token = ""
	assert.Error(t, c.Validate())
}

func TestLegacyOptionsAliasing(t *testing.T) {
	viper.Reset()
	viper.Set("source", "test")
	UpdateLegacyString("target", "source")
	assert.Equal(t, "", viper.GetString("target"))
	UpdateLegacyString("source", "target")
	assert.Equal(t, "test", viper.GetString("target"))
	viper.Set("source", "newTest")
	assert.Equal(t, "test", viper.GetString("target"))
}
