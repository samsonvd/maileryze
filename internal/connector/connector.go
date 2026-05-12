package connector

import (
	"time"
)

type EmailSender struct {
	Name    string
	Address string
}

type UnsubscribeMechanism struct {
	Email string // mailto: address from List-Unsubscribe header
	URL   string // HTTP URL from List-Unsubscribe header
}

type ProviderDetails[T any] struct {
	Identifier string
	Data       T
}

type EmailContent[T any] struct {
	Subject     string
	Sender      EmailSender
	Unsubscribe UnsubscribeMechanism
	Provider    ProviderDetails[T]
}

type Connector[T any] interface {
	Login() error
	Fetch(start, end time.Time) ([]EmailContent[T], error)
}

type NilConnector struct{}

func (c *NilConnector) Login() error {
	return nil
}

func (c *NilConnector) Fetch(start, end time.Time) ([]EmailContent[any], error) {
	return []EmailContent[any]{}, nil
}
