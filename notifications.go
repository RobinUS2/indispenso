package main

import "sync"

const (
	NEW_CONSENSUS  NotificationType = "New Consensus Request"
	EXECUTION_DONE NotificationType = "Execution done"
)

type NotificationService interface {
	GetConsumer() func(*Message)
}

type NotificationType string

type NotificationServiceConfig interface {
	IsValid() error
	IsEnabled() bool
	GetService() (NotificationService, error)
	GetName() string
}

type Message struct {
	Content string
	Type    NotificationType
	Url     string
}

func newMessage(content string, nType NotificationType) *Message {
	return &Message{Content: content, Type: nType}
}

type MessageReceiver struct {
	channel  chan *Message
	consumer func(*Message)
}

func (r *MessageReceiver) run() {
	for {
		select {
		case msg := <-r.channel:
			r.consumer(msg)
		default:
		}
	}
}

type NotificationManager struct {
	muxReceivers     sync.RWMutex
	listeners        []*MessageReceiver
	sendOut          chan *Message
	registerReceiver chan *MessageReceiver
}

func newNotificationManager() *NotificationManager {
	b := &NotificationManager{
		listeners: []*MessageReceiver{},
		sendOut:   make(chan *Message, 100),
	}

	go b.dispatcher()
	return b
}

func (m *NotificationManager) dispatcher() {
	for {
		select {
		case value := <-m.sendOut:
			for _, listener := range m.listeners {
				select {
				case listener.channel <- value:
				default:
					log.Printf("Ommit passing notification to listener, channel full")
				}
			}
		default:
		}
	}
}

func (m *NotificationManager) registerWithChannelSize(service NotificationService, channelSize int) {
	msgReceiver := &MessageReceiver{channel: make(chan *Message, channelSize), consumer: service.GetConsumer()}
	go msgReceiver.run()

	m.muxReceivers.Lock()
	defer m.muxReceivers.Unlock()
	m.listeners = append(m.listeners, msgReceiver)
}

func (m *NotificationManager) register(service NotificationService) {
	m.registerWithChannelSize(service, 1000)
}

func (m *NotificationManager) Notify(msg *Message) {
	m.sendOut <- msg
}
