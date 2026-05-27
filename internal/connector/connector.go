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
	Email string // mailto: address from List-Unsubscribe
	URL   string // HTTP URL from List-Unsubscribe
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

// WritableConnector extends Connector with inbox-management operations.
type WritableConnector interface {
	Connector[any]
	// TrashAllFrom searches Gmail live for all messages from senderAddress and
	// moves them to Trash. Returns the number of messages trashed.
	TrashAllFrom(ctx context.Context, senderAddress string) (int, error)
	// TrashMessages moves specific messages (by provider ID) to Trash.
	TrashMessages(ctx context.Context, messageIDs []string) error
	// SendEmail sends a plain-text email from the authenticated account.
	SendEmail(ctx context.Context, to, subject, body string) error
}

type NilConnector struct{}

func (c *NilConnector) Login() error { return nil }

func (c *NilConnector) Fetch(ctx context.Context, start, end time.Time) <-chan Result[EmailContent[any]] {
	ch := make(chan Result[EmailContent[any]])
	close(ch)
	return ch
}

func (c *NilConnector) TrashAllFrom(_ context.Context, _ string) (int, error) { return 0, nil }
func (c *NilConnector) TrashMessages(_ context.Context, _ []string) error      { return nil }
func (c *NilConnector) SendEmail(_ context.Context, _, _, _ string) error      { return nil }
