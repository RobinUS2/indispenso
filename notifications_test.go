package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

const DONE_MESSAGE = NotificationType("done")

type TestNotificationService struct {
	messages    []*Message
	allReceived chan bool
}

func (s *TestNotificationService) GetConsumer() func(*Message) {
	return func(msg *Message) {
		s.messages = append(s.messages, msg)
		if msg.Type == DONE_MESSAGE {
			s.allReceived <- true
		}
	}
}

func newTestNotificationService() *TestNotificationService {
	return &TestNotificationService{messages: []*Message{}, allReceived: make(chan bool)}
}

func TestAbleToSendMessageAcrossManager(t *testing.T) {
	m := newNotificationManager()
	ts := newTestNotificationService()
	m.register(ts)

	m.Notify(&Message{Content: "test", Type: DONE_MESSAGE})

	<-ts.allReceived

	assert.Len(t, ts.messages, 1)
	assert.Equal(t, "test", ts.messages[0].Content)
}

func TestMultipleReceivers(t *testing.T) {
	m := newNotificationManager()
	ts1 := newTestNotificationService()
	ts2 := newTestNotificationService()
	m.register(ts1)
	m.register(ts2)

	m.Notify(&Message{Content: "test", Type: DONE_MESSAGE})

	<-ts1.allReceived
	<-ts2.allReceived

	assert.Len(t, ts1.messages, 1)
	assert.Equal(t, "test", ts1.messages[0].Content)
	assert.Len(t, ts2.messages, 1)
	assert.Equal(t, "test", ts2.messages[0].Content)
}

func TestMessageCreation(t *testing.T) {
	msg := newMessage("testContent", "testType")

	assert.Equal(t, NotificationType("testType"), msg.Type)
	assert.Equal(t, "testContent", msg.Content)
}

func BenchmarkNotificationSpeed(b *testing.B) {
	m := newNotificationManager()
	ts := newTestNotificationService()

	m.registerWithChannelSize(ts, b.N)

	b.ResetTimer()
	for i := 0; i < b.N-1; i++ { // b.N-1 due to we send final message as signaling one
		m.Notify(&Message{Content: fmt.Sprintf("%d-test", i), Url: "test", Type: NotificationType("Test")})
	}

	m.Notify(&Message{Content: "test", Url: "test", Type: DONE_MESSAGE})
	<-ts.allReceived

}
