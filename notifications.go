package main

type NotificationService interface {
	SetReceiver(MessageReceiver)
}

type NotificationType string

type Message struct {
	Content string
	Type    NotificationType
}

type MessageReceiver struct {
	channel  chan BroadcastMessage
	consumer func(msg Message)
}

type BroadcastMessage struct {
	channel chan BroadcastMessage
	value   Message
}

func newMessage(content string, nType NotificationType) *Message {
	return &Message{Content: content, Type: nType}
}

type NotificationManager struct {
	listenChan chan chan (chan BroadcastMessage)
	sendOut    chan Message
}

func newNotificationManager() NotificationManager {
	listenChan := make(chan chan (chan BroadcastMessage))
	sendChan := make(chan Message)

	b := NotificationManager{
		listenChan: listenChan,
		sendOut:    sendChan,
	}

	go b.dispatcher()
	return b
}

func (m *NotificationManager) dispatcher() {
	currentChannel := make(chan BroadcastMessage, 1)
	for {
		select {
		case value := <-m.sendOut:
			if &value == nil {
				currentChannel <- BroadcastMessage{}
				return
			}
			msgToBroadcast := BroadcastMessage{
				channel: make(chan BroadcastMessage, 1),
				value:   value,
			}
			currentChannel <- msgToBroadcast
			currentChannel = msgToBroadcast.channel
		case result := <-m.listenChan:
			result <- currentChannel
		}
	}
}

func (m *NotificationManager) register(service NotificationService) {
	channel := make(chan chan BroadcastMessage, 0)
	m.listenChan <- channel
	service.SetReceiver(MessageReceiver{channel: <-channel, consumer: func(msg Message) {}})
}

func (m *NotificationManager) Notify(msg Message) {
	m.sendOut <- msg
}

func (r *MessageReceiver) Read() Message {
	broadcastMsg := <-r.channel
	msg := broadcastMsg.value

	r.channel <- broadcastMsg
	r.channel = broadcastMsg.channel
	return msg
}

func (r *MessageReceiver) consumeMsg() {
	for msg := r.Read(); &msg != nil; msg = r.Read() {
		go r.consumeMsg()
		r.consumer(msg)
	}
}

func (r *MessageReceiver) Consume(consumer func(msg Message)) {
	r.consumer = consumer
	go r.consumeMsg()
}
