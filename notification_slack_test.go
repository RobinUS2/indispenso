package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testSlack = newSlackNotify(&SlackNotifyConfig{Token: "testToken", NotifyAs: "testUser", ChannelName: "testChannel"})

func TestSlackStringMessage(t *testing.T) {
	msgString, err := testSlack.MsgString(&Message{Content: "testContent", Type: "Test Type", Url: "http://example.com/a"})

	assert.NoError(t, err)
	assert.Contains(t, msgString, "http://example.com/a")
	assert.Contains(t, msgString, "testContent")
	assert.Contains(t, msgString, "Test Type")
}

func TestSlackOptsMessage(t *testing.T) {
	msgOptions, err := testSlack.MsgOptions(&Message{Content: "testContent", Type: "Test Type"})

	assert.NoError(t, err)
	assert.Equal(t, "testUser", msgOptions.Username)
	assert.False(t, msgOptions.AsUser)
}

func TestSlackImplementsNotificationService(t *testing.T) {
	assert.Implements(t, (*NotificationService)(nil), testSlack)
}

func TestSlackConfigImplementsNotificationServiceConfig(t *testing.T) {
	assert.Implements(t, (*NotificationServiceConfig)(nil), &SlackNotifyConfig{})
}

func TestSlackEnabledConfig(t *testing.T) {
	config := &SlackNotifyConfig{}

	assert.False(t, config.IsEnabled())
	config.Token = "test"
	assert.True(t, config.IsEnabled())
}

func TestSlackConfigValidation(t *testing.T) {
	config := &SlackNotifyConfig{}
	assert.Error(t, config.IsValid())

	config.Token = "test"
	config.ChannelName = "test"
	config.NotifyAs = "test"

	assert.NoError(t, config.IsValid())
}

func TestEmptyChannelName(t *testing.T) {
	config := &SlackNotifyConfig{Token: "tets", NotifyAs: "test"}
	assert.EqualError(t, config.IsValid(), "CannelName cannot be empty")
}

func TestEmptyToken(t *testing.T) {
	config := &SlackNotifyConfig{ChannelName: "tets", NotifyAs: "test"}
	assert.EqualError(t, config.IsValid(), "Token cannot be empty")
}

func TestSlackServiceName(t *testing.T) {
	config := &SlackNotifyConfig{}
	assert.Equal(t, "Slack", config.GetName())
}

func TestSlackServiceCreation(t *testing.T) {
	config := &SlackNotifyConfig{Token: "testToken", NotifyAs: "testUser", ChannelName: "testChannel"}
	service, err := config.GetService()
	assert.NoError(t, err)
	assert.IsType(t, &SlackNotify{}, service)
}
