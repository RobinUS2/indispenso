package main

import (
	"errors"
	"fmt"
	"github.com/bluele/slack"
)

type SlackNotify struct {
	slackApi   *slack.Slack
	Config     *SlackNotifyConfig
	msgChannel chan chan *Message
}

type SlackNotifyConfig struct {
	Token       string
	ChannelName string
	NotifyAs    string
}

func (c *SlackNotifyConfig) IsValid() error {
	if len(c.ChannelName) < 1 {
		return errors.New("CannelName cannot be empty")
	}

	if len(c.Token) < 1 {
		return errors.New("Token cannot be empty")
	}
	return nil
}

func (c *SlackNotifyConfig) IsEnabled() bool {
	return len(c.Token) > 0
}

func (c *SlackNotifyConfig) GetService() (NotificationService, error) {
	return newSlackNotify(c), nil
}

func (c *SlackNotifyConfig) GetName() string {
	return "Slack"
}

func newSlackNotify(conf *SlackNotifyConfig) *SlackNotify {
	sn := &SlackNotify{
		slackApi: slack.New(conf.Token),
		Config:   conf,
	}
	return sn
}

func (s *SlackNotify) GetConsumer() func(*Message) {
	return func(msg *Message) {
		if err := s.SendMessage(msg); err != nil {
			log.Printf("Cannot send message due to: %s", err)
		}
	}
}

func (s *SlackNotify) SendMessage(msg *Message) error {
	content, err := s.MsgString(msg)
	if err != nil {
		return err
	}

	opts, err := s.MsgOptions(msg)
	if err != nil {
		return err
	}

	channel, err := s.slackApi.FindChannelByName(s.Config.ChannelName)
	if err != nil {
		return err
	}

	if err := s.slackApi.ChatPostMessage(channel.Id, content, opts); err != nil {
		return err
	}

	return nil
}

func (s *SlackNotify) MsgString(msg *Message) (string, error) {
	return fmt.Sprintf("%s: %s \n see more here: <%s>", msg.Type, msg.Content, msg.Url), nil
}

func (s *SlackNotify) MsgOptions(msg *Message) (*slack.ChatPostMessageOpt, error) {
	return &slack.ChatPostMessageOpt{
		AsUser:   false,
		Username: s.Config.NotifyAs,
	}, nil
}
