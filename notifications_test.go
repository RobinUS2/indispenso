package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type TestNotificationService struct {
	messages []Message
}

func (s *TestNotificationService) SetReceiver(receiver MessageReceiver) {
	receiver.Consume(func(msg Message) {
		s.messages = append(s.messages, msg)
	})
}

func newTestNotificationService() *TestNotificationService {
	return &TestNotificationService{messages: []Message{}}
}

func TestAbleToSendMessageAcrossManager(t *testing.T) {
	m := newNotificationManager()
	ts := newTestNotificationService()
	m.register(ts)

	m.Notify(Message{Content: "test", Type: "test"})

	assert.Len(t, ts.messages, 1)
	assert.Equal(t, "test", ts.messages[0].Content)
}

func TestMultipleReceivers(t *testing.T) {
	m := newNotificationManager()
	ts1 := newTestNotificationService()
	ts2 := newTestNotificationService()
	m.register(ts1)
	m.register(ts2)

	m.Notify(Message{Content: "test", Type: "test"})

	time.Sleep(1 * time.Second)

	assert.Len(t, ts1.messages, 1)
	assert.Equal(t, "test", ts1.messages[0].Content)
	assert.Len(t, ts2.messages, 1)
	assert.Equal(t, "test", ts2.messages[0].Content)
}
