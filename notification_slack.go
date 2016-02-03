package main

import (
	"fmt"
	"github.com/bluele/slack"
)

type SlackNotify struct {
	slackApi  *slack.Slack
	ChannelId string
	NotifyAs  string
}

func newSlackNotify(token string, channelId string, notifyAs string) *SlackNotify {
	return &SlackNotify{
		slackApi:  slack.New(token),
		ChannelId: channelId,
		NotifyAs:  notifyAs,
	}
}

func (s *SlackNotify) SetMsgReceiver(receiver MessageReceiver) {
	receiver.Consume(s.SendMessage)
}

func (s *SlackNotify) SendMessage(msg Message) {
	content, err := s.MsgString(msg)
	if err != nil {
		log.Printf("Cannot send notification to Slack due to: %s", err)
		return
	}

	opts, err := s.MsgOptions(msg)
	if err != nil {
		log.Printf("Cannot send notification to Slack due to: %s", err)
		return
	}

	if err := s.slackApi.ChatPostMessage(s.ChannelId, content, opts); err != nil {
		log.Printf("Cannot send notification to Slack due to: %s", err)
	}
}

func (s *SlackNotify) MsgString(msg Message) (string, error) {
	return fmt.Sprintf("%s: %s", msg.Type, msg.Content), nil
}

func (s *SlackNotify) MsgOptions(msg Message) (*slack.ChatPostMessageOpt, error) {
	return &slack.ChatPostMessageOpt{
		AsUser:   true,
		Username: s.NotifyAs,
	}, nil
}
