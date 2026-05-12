package connector

import (
	"maileryze/internal/types"
	"time"
)

type EmailSender struct {
	Name    string
	Address string
}

type UnsubscribeMechanism struct{}

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

func (c *NilConnector) Fetch(start, end time.Time) ([]EmailContent[string], error) {
	return []EmailContent[string]{}, nil
}

func ConnectorFactory(provider types.Provider) *Connector[any] {
	if provider == types.ProviderGmail {

	}
	return &NilConnector{}
}
