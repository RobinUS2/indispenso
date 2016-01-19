package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitialisingWithCorrectAddress(t *testing.T) {
	config := &LdapConfig{ServerAddress: "ldaps://123.test.local:900"}
	err := config.Init()

	assert.NoError(t, err)
	assert.Equal(t, true, config.isTLS)
	assert.Equal(t, "123.test.local", config.host)
	assert.Equal(t, "900", config.port)
}

func TestInitialisingWithoutPort(t *testing.T) {
	config := &LdapConfig{ServerAddress: "ldaps://123.test.local"}
	err := config.Init()

	assert.NoError(t, err)
	assert.Equal(t, true, config.isTLS)
	assert.Equal(t, "123.test.local", config.host)
	assert.Equal(t, "636", config.port)
}

func TestInvalidProtocol(t *testing.T) {
	config := &LdapConfig{ServerAddress: "http://123.test.local"}
	err := config.Init()

	assert.EqualError(t, err, "Cannot parse srever address")
}
