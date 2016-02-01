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

	assert.EqualError(t, err, "Cannot parse server address")
}

func TestUserFilterCreation(t *testing.T) {
	config := &LdapConfig{UserSearchFilter: "test=%s"}
	userFilter := config.GetUserSearchFilter("testUser")

	assert.Equal(t, "(&(test=testUser))", userFilter)
}

func TestReinitializeLdapConfig(t *testing.T) {
	config := &LdapConfig{ServerAddress: "ldaps://123.test.local"}
	config.Init()
	assert.NoError(t, config.Init())
}

func TestDefaultPortForNotTLSConnection(t *testing.T) {
	config := &LdapConfig{ServerAddress: "ldap://123.test.local"}
	err := config.Init()
	assert.NoError(t, err)
	assert.Equal(t, "389", config.port)
}

func TestGetProperAddress(t *testing.T) {
	config := &LdapConfig{host: "test", port: "345"}
	assert.Equal(t, "test:345", config.GetAddress())
}

func TestAttributesContainsAllMandatoryItems(t *testing.T) {
	config := &LdapConfig{EmailAttr: "email", Attributes: []string{}}
	attributes := config.GetAttributes()
	assert.Contains(t, attributes, "dn")
	assert.Contains(t, attributes, "email")
}

func TestAllCustomAttributesShouldBeUsed(t *testing.T) {
	config := &LdapConfig{EmailAttr: "email", Attributes: []string{"test", "test2"}}
	attributes := config.GetAttributes()
	assert.Contains(t, attributes, "test")
	assert.Contains(t, attributes, "test2")
}
