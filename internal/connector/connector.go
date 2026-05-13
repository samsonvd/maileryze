package connector

import (
	"context"
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
	ReceivedAt  time.Time
}

type Result[T any] struct {
	Value T
	Err   error
}

type Connector[T any] interface {
	Login() error
	Fetch(ctx context.Context, start, end time.Time) <-chan Result[EmailContent[T]]
}

type NilConnector struct{}

func (c *NilConnector) Login() error {
	return nil
}

func (c *NilConnector) Fetch(ctx context.Context, start, end time.Time) <-chan Result[EmailContent[any]] {
	ch := make(chan Result[EmailContent[any]])
	close(ch)
	return ch
}
