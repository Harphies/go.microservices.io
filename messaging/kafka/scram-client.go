package kafka

import (
	"crypto/sha512"
	"github.com/xdg-go/scram"
)

var (
	SHA512 scram.HashGeneratorFcn = sha512.New
)

type Client struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

// NewKafkaClient establish a new SCRAM connection
func NewKafkaClient() *Client {
	return &Client{HashGeneratorFcn: SHA512}
}

func (c *Client) Begin(userName, password, authzID string) (err error) {
	c.Client, err = c.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	c.ClientConversation = c.Client.NewConversation()
	return nil
}

func (c *Client) Setup(challenge string) (response string, err error) {
	return c.ClientConversation.Step(challenge)
}

func (c *Client) Done() bool {
	return c.ClientConversation.Done()
}
